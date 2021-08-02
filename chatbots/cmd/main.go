package main

import (
	"context"
	"datafuselabs/test-infra/chatbots/hook"
	"datafuselabs/test-infra/chatbots/utils"
	"github.com/google/uuid"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jnovack/flag"
	"github.com/rs/zerolog/log"
)

const (
	LeassLockName      = "datafuse-chatbot-lock"
	LeaseLockNamespace = "kube-system"
)

var (
	GithubToken          string
	WebhookToken         string
	Address              string
	Region               string
	Bucket               string
	Endpoint             string
	TemplateDir          string
	StaticDir            string
	EnableLeaderElection bool
)

func init() {
	flag.StringVar(&GithubToken, "github-token", "", "chatbot github token")
	flag.StringVar(&WebhookToken, "webhook-token", "", "webhook token for chatbot server")
	flag.StringVar(&Address, "address", "", "address that chatbot server binds to")
	flag.StringVar(&Region, "region", "", "S3 Storage Region")
	flag.StringVar(&Bucket, "bucket", "", "S3 Storage bucket")
	flag.StringVar(&Endpoint, "endpoint", "", "S3 storage endpoint")
	flag.StringVar(&TemplateDir, "template-dir", "", "dashboard template dir")
	flag.StringVar(&StaticDir, "static-dir", "", "dashboard static file dir")
	flag.BoolVar(&EnableLeaderElection, "enable-leader-election", false, "configure leader election for k8s HA")

}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}


// Set up Kubernetes client using kubeconfig (or in-cluster config if no file provided)
func clientSetup(kubeconfig string) (clientset *client.Clientset, err error) {
	config, err := buildConfig(kubeconfig)
	if err != nil {
		return
	}
	clientset, err = client.NewForConfig(config)
	return
}

func main() {
	flag.Parse()
	if !strings.HasPrefix(Endpoint, "http://") && !strings.HasPrefix(Endpoint, "https://") {
		Endpoint = "https://" + Endpoint
	}
	cfg := hook.NewConfig(
		&utils.FileStorage{
			BasePath: "./tmp",
		},
		context.Background(),
		log.Logger,
		GithubToken,
		WebhookToken,
		Address,
		Region, Bucket, Endpoint,
		TemplateDir,
		StaticDir,
	)
	server := hook.NewServer(cfg)
	// run local
	if !EnableLeaderElection {
		server.Start()
		return
	}

	// needs to run on a k8s cluster
	id := uuid.New().String()
	clientSet, err := clientSetup("")
	if err != nil {
		log.Error().Msgf("error on k8s client setup %s", err.Error())
		return
	}
	stopCh := make(chan struct{})
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      LeassLockName,
			Namespace: LeaseLockNamespace,
		},
		Client: clientSet.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}
	leadelectionCallback := leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			// once leader elected it should taint all nodes at first to prevent race condition
			log.Log().Msgf("Start to run server on %s", id)
			server.Start()
		},
		OnStoppedLeading: func() {
			// when leader failed, log leader failure and restart leader election
			log.Log().Msgf("leader lost: %s", id)
		},
		OnNewLeader: func(identity string) {
			// we're notified when new leader elected
			if identity == id {
				// I just got the lock
				return
			}
			log.Log().Msgf("new leader elected: %s", identity)
		},
	}
	func(stopCh <-chan struct{}) {
		for {
			ctx, cancel := context.WithCancel(context.Background())
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-ch
				log.Log().Msgf("Received termination, signaling shutdown")
				cancel()
			}()
			leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
				Lock:            lock,
				ReleaseOnCancel: true,
				LeaseDuration:   60 * time.Second,
				RenewDeadline:   15 * time.Second,
				RetryPeriod:     5 * time.Second,
				Callbacks:       leadelectionCallback,
			})
			select {
			case <-stopCh:
				return
			default:
				cancel()
				log.Error().Msgf("leader election lost due to exception happened")
			}
		}
	}(stopCh)
}
