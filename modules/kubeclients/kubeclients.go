package kubeclients

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"pkg.package-operator.run/cardboard/kubeutils/kubemanifests"
	"pkg.package-operator.run/cardboard/kubeutils/wait"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Creates a bunch of useful kubernetes client interfaces to interact with clusters.
// Needs to be initialized with a call to .Run(), can be used directly as a Dependency.
type KubeClients struct {
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	CtrlClient client.Client
	Waiter     *wait.Waiter

	kubeconfigPath string
}

func init() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
}

var defaultSchemeBuilder runtime.SchemeBuilder = runtime.SchemeBuilder{
	clientgoscheme.AddToScheme,
	apiextensionsv1.AddToScheme,
}

type KubeClientsOption interface {
	ApplyToKubeClients(kc *KubeClients)
}

func NewKubeClients(kubeconfigPath string, opts ...KubeClientsOption) *KubeClients {
	kc := &KubeClients{
		Scheme:         runtime.NewScheme(),
		kubeconfigPath: kubeconfigPath,
	}
	for _, opt := range opts {
		opt.ApplyToKubeClients(kc)
	}

	return kc
}

func (kc *KubeClients) Run(_ context.Context) error {
	// Add default schemes
	if err := defaultSchemeBuilder.AddToScheme(kc.Scheme); err != nil {
		return fmt.Errorf("adding defaults to scheme: %w", err)
	}

	var err error
	// Create RestConfig
	kc.RestConfig, err = clientcmd.BuildConfigFromFlags("", kc.kubeconfigPath)
	if err != nil {
		return fmt.Errorf("getting rest.Config from kubeconfig: %w", err)
	}

	// Create Controller Runtime Client
	kc.CtrlClient, err = client.New(kc.RestConfig, client.Options{
		Scheme: kc.Scheme,
	})

	if err != nil {
		return fmt.Errorf("creating new ctrl client: %w", err)
	}

	kc.Waiter = wait.NewWaiter(
		kc.CtrlClient, kc.Scheme)
	return nil
}

func (kc *KubeClients) ID() string {
	return fmt.Sprintf("pkg.package-operator.run/cardboard/modules.KubeClients{kubeconfigPath:%s}", kc.kubeconfigPath)
}

// Load kube objects from a list of http urls,
// create these objects and wait for them to be ready.
func (kc *KubeClients) CreateAndWaitFromHTTP(
	ctx context.Context, urls []string,
	opts ...wait.Option,
) error {
	var client http.Client
	var objects []unstructured.Unstructured
	for _, url := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("getting %q: %w", url, err)
		}
		defer resp.Body.Close()

		var content bytes.Buffer
		if _, err := io.Copy(&content, resp.Body); err != nil {
			return fmt.Errorf("reading response %q: %w", url, err)
		}

		objs, err := kubemanifests.LoadKubernetesObjectsFromBytes(content.Bytes())
		if err != nil {
			return fmt.Errorf("loading objects from %q: %w", url, err)
		}

		objects = append(objects, objs...)
	}

	return kc.createObjectsFromSource(ctx, "http", objects, opts...)
}

// Load kube objects from a list of files,
// create these objects and wait for them to be ready.
func (kc *KubeClients) CreateAndWaitFromFiles(
	ctx context.Context, files []string,
	opts ...wait.Option,
) error {
	var objects []unstructured.Unstructured
	for _, file := range files {
		objs, err := kubemanifests.LoadKubernetesObjectsFromFile(file)
		if err != nil {
			return fmt.Errorf("loading objects from file %q: %w", file, err)
		}

		objects = append(objects, objs...)
	}

	return kc.createObjectsFromSource(ctx, "files", objects, opts...)
}

// Load kube objects from a list of folders,
// create these objects and wait for them to be ready.
func (kc *KubeClients) CreateAndWaitFromFolders(
	ctx context.Context, folders []string,
	opts ...wait.Option,
) error {
	var objects []unstructured.Unstructured
	for _, folder := range folders {
		objs, err := kubemanifests.LoadKubernetesObjectsFromFolder(folder)
		if err != nil {
			return fmt.Errorf("loading objects from folder %q: %w", folder, err)
		}

		objects = append(objects, objs...)
	}

	return kc.createObjectsFromSource(ctx, "folders", objects, opts...)
}

func (kc *KubeClients) createObjectsFromSource(
	ctx context.Context, source string,
	objects []unstructured.Unstructured, opts ...wait.Option,
) error {
	for i := range objects {
		if err := kc.CreateAndWaitForReadiness(ctx, &objects[i], opts...); err != nil {
			return fmt.Errorf("creating from %s: %w", source, err)
		}
	}
	return nil
}

// Creates the given objects and waits for them to be considered ready.
func (kc *KubeClients) CreateAndWaitForReadiness(
	ctx context.Context, object client.Object,
	opts ...wait.Option,
) error {
	if err := kc.CtrlClient.Create(ctx, object); err != nil &&
		!apimachineryerrors.IsAlreadyExists(err) {
		gvk := object.GetObjectKind().GroupVersionKind()
		return fmt.Errorf("creating object: %s/%s/%s %s/%s: %w",
			gvk.Group,
			gvk.Version,
			gvk.Kind,
			object.GetNamespace(), object.GetName(), err)
	}

	if err := kc.Waiter.WaitForReadiness(ctx, object, opts...); err != nil {
		var unknownTypeErr *wait.UnknownTypeError
		if errors.As(err, &unknownTypeErr) {
			// A lot of types don't require waiting for readiness,
			// so we should not error in cases when object types
			// are not registered for the generic wait method.
			return nil
		}

		return fmt.Errorf("waiting for object: %w", err)
	}
	return nil
}
