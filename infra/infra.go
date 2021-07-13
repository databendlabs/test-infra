package main // import "github.com/prometheus/test-infra/infra"

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"datafuselabs/test-infra/pkg/provider"
	kind "datafuselabs/test-infra/pkg/provider/kind"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	dr := provider.NewDeploymentResource()

	app := kingpin.New(filepath.Base(os.Args[0]), "The prometheus/test-infra deployment tool")
	app.HelpFlag.Short('h')
	app.Flag("file", "yaml file or folder  that describes the parameters for the object that will be deployed.").
		Short('f').
		ExistingFilesOrDirsVar(&dr.DeploymentFiles)
	app.Flag("vars", "When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.").
		Short('v').
		StringMapVar(&dr.FlagDeploymentVars)

	k := kind.New(dr)
	k8sKIND := app.Command("kind", `Kubernetes In Docker (KIND) provider - https://kind.sigs.k8s.io/docs/user/quick-start/`).
		Action(k.SetupDeploymentResources)

	k8sKIND.Command("info", "kind info -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(k.GetDeploymentVars)

	//Cluster operations.
	k8sKINDCluster := k8sKIND.Command("cluster", "manage KIND clusters").
		Action(k.KINDDeploymentsParse)
	k8sKINDCluster.Command("create", "kind cluster create -f File -v PR_NUMBER:$PR_NUMBER -v CLUSTER_NAME:$CLUSTER_NAME").
		Action(k.ClusterCreate)
	k8sKINDCluster.Command("delete", "kind cluster delete -f File -v PR_NUMBER:$PR_NUMBER -v CLUSTER_NAME:$CLUSTER_NAME").
		Action(k.ClusterDelete)

	// K8s resource operations.
	k8sKINDResource := k8sKIND.Command("resource", `Apply and delete different k8s resources - deployments, services, config maps etc.`).
		Action(k.NewK8sProvider).
		Action(k.K8SDeploymentsParse)
	k8sKINDResource.Command("apply", "kind resource apply -f manifestsFileOrFolder -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(k.ResourceApply)
	k8sKINDResource.Command("delete", "kind resource delete -f manifestsFileOrFolder -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(k.ResourceDelete)


	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

}