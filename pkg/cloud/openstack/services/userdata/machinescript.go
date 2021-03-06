/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package userdata

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"path"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/options"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/provider"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/deployer"
	"sigs.k8s.io/cluster-api/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
	"text/template"
	"time"

	"fmt"

	"encoding/json"
	clconfig "github.com/coreos/container-linux-config-transpiler/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tokenapi "k8s.io/cluster-bootstrap/token/api"
	tokenutil "k8s.io/cluster-bootstrap/token/util"
	providerv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/bootstrap"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	apierrors "sigs.k8s.io/cluster-api/pkg/errors"
)

const (
	UserDataKey          = "userData"
	DisableTemplatingKey = "disableTemplating"
	PostprocessorKey     = "postprocessor"
)

type setupParams struct {
	Machine *clusterv1.Machine

	CACert           string
	CAKey            string
	EtcdCACert       string
	EtcdCAKey        string
	FrontProxyCACert string
	FrontProxyCAKey  string
	SaCert           string
	SaKey            string

	KubeadmConfig string
}

func GetUserData(controllerClient client.Client, kubeClient kubernetes.Interface, machineProviderSpec *providerv1.OpenstackProviderSpec, cluster *clusterv1.Cluster, machine *clusterv1.Machine) (string, error) {

	// get machine startup script
	var ok bool
	var disableTemplating bool
	var postprocessor string
	var postprocess bool
	var err error

	var userData []byte
	if machineProviderSpec.UserDataSecret != nil {
		namespace := machineProviderSpec.UserDataSecret.Namespace
		if namespace == "" {
			namespace = machine.Namespace
		}

		if machineProviderSpec.UserDataSecret.Name == "" {
			return "", fmt.Errorf("UserDataSecret name must be provided")
		}

		userDataSecret, err := kubeClient.CoreV1().Secrets(namespace).Get(machineProviderSpec.UserDataSecret.Name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}

		userData, ok = userDataSecret.Data[UserDataKey]
		if !ok {
			return "", fmt.Errorf("machine's userdata secret %v in namespace %v did not contain key %v", machineProviderSpec.UserDataSecret.Name, namespace, UserDataKey)
		}

		_, disableTemplating = userDataSecret.Data[DisableTemplatingKey]

		var p []byte
		p, postprocess = userDataSecret.Data[PostprocessorKey]

		postprocessor = string(p)
	} else if options.UserDataFolder != "" {
		userData, err = ioutil.ReadFile(path.Join(options.UserDataFolder, fmt.Sprintf("%s.yaml", machine.Name)))
		if err != nil {
			return "", fmt.Errorf("could not load local userdata files: %v", err)
		}
		postprocessor = options.UserDataPostprocessor
		if postprocessor != "" {
			postprocess = true
		}
	}

	var userDataRendered string
	if len(userData) > 0 && !disableTemplating {
		isNodeJoin, err := isNodeJoin(controllerClient, kubeClient, cluster, machine)
		if err != nil {
			return "", apierrors.CreateMachine("error creating Openstack instance: %v", err)
		}

		var bootstrapToken string
		if isNodeJoin {
			klog.Info("Creating bootstrap token")
			bootstrapToken, err = createBootstrapToken(controllerClient)
			if err != nil {
				return "", apierrors.CreateMachine("error creating Openstack instance: %v", err)
			}
		}

		userDataRendered, err = startupScript(cluster, machine, machineProviderSpec, string(userData), bootstrapToken)
		if err != nil {
			return "", apierrors.CreateMachine("error creating Openstack instance: %v", err)
		}
		if util.IsControlPlaneMachine(machine) && isNodeJoin {
			// A little bit hacky but maybe good enough until v1alpha2. The alternative would be to template
			// either the kubeadm command or the whole kubeadm service file. But I think the 2nd option would
			// reduce the flexibility too much.
			userDataRendered = strings.ReplaceAll(userDataRendered, "kubeadm init", "kubeadm join")
		}
	} else {
		userDataRendered = string(userData)
	}

	if postprocess {
		switch postprocessor {
		// Postprocess with the Container Linux ct transpiler.
		case "ct":
			clCfg, ast, report := clconfig.Parse([]byte(userDataRendered))
			if len(report.Entries) > 0 {
				return "", fmt.Errorf("Postprocessor error: %s", report.String())
			}

			ignCfg, report := clconfig.Convert(clCfg, "openstack-metadata", ast)
			if len(report.Entries) > 0 {
				return "", fmt.Errorf("Postprocessor error: %s", report.String())
			}

			ud, err := json.Marshal(&ignCfg)
			if err != nil {
				return "", fmt.Errorf("Postprocessor error: %s", err)
			}

			userDataRendered = string(ud)

		default:
			return "", fmt.Errorf("Postprocessor error: unknown postprocessor: '%s'", postprocessor)
		}
	}
	return userDataRendered, nil
}

func isNodeJoin(controllerClient client.Client, kubeClient kubernetes.Interface, cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {

	// Worker machines always join
	if !util.IsControlPlaneMachine(machine) {
		klog.Infof("Worker machine %s is joining the cluster\n", machine.Name)
		return true, nil
	}

	// Get control plane machines and return false if none found
	controlPlaneMachines, err := getControlPlaneMachines(controllerClient)
	if err != nil {
		return false, apierrors.CreateMachine("error retrieving control plane machines: %v", err)
	}
	if len(controlPlaneMachines) == 0 {
		klog.Infof("Could not find control plane machine: creating first control plane machine %s\n", machine.Name)
		return false, nil
	}

	// Get control plane machine instances and return false if none found
	osProviderClient, clientOpts, err := provider.NewClientFromCluster(kubeClient, cluster)
	if err != nil {
		return false, err
	}
	computeService, err := compute.NewService(osProviderClient, clientOpts)
	if err != nil {
		return false, err
	}
	instanceList, err := computeService.GetInstanceList(&compute.InstanceListOpts{})
	if err != nil {
		return false, err
	}
	if len(instanceList) == 0 {
		klog.Infof("Could not find control plane machine: creating first control plane machine %s\n", machine.Name)
		return false, nil
	}

	for _, controlPlaneMachine := range controlPlaneMachines {
		for _, instance := range instanceList {
			if controlPlaneMachine.Name == instance.Name {
				klog.Infof("Found control plane machine %s: control plane machine %s is joining the cluster\n", controlPlaneMachine.Name, machine.Name)
				return true, nil
			}
		}
	}
	klog.Infof("Could not find control plane machine: creating first control plane machine %s\n", machine.Name)
	return false, nil
}

func getControlPlaneMachines(controllerClient client.Client) ([]*clusterv1.Machine, error) {
	var controlPlaneMachines []*clusterv1.Machine
	msList := &clusterv1.MachineList{}
	listOptions := &client.ListOptions{}
	err := controllerClient.List(context.TODO(), listOptions, msList)
	if err != nil {
		return nil, fmt.Errorf("error retrieving machines: %v", err)
	}
	for _, m := range msList.Items {
		if util.IsControlPlaneMachine(&m) {
			// we need DeepCopy because if we append the Pointer it will all be
			// the same machine
			controlPlaneMachines = append(controlPlaneMachines, m.DeepCopy())
		}
	}
	return controlPlaneMachines, nil
}

func createBootstrapToken(controllerClient client.Client) (string, error) {
	token, err := tokenutil.GenerateBootstrapToken()
	if err != nil {
		return "", err
	}

	expiration := time.Now().UTC().Add(options.TokenTTL)
	tokenSecret, err := bootstrap.GenerateTokenSecret(token, expiration)
	if err != nil {
		panic(fmt.Sprintf("unable to create token. there might be a bug somwhere: %v", err))
	}

	kubeClient, err := getWorkloadClusterKubeClient(controllerClient)
	if err != nil {
		return "", err
	}

	err = kubeClient.Create(context.TODO(), tokenSecret)
	if err != nil {
		return "", err
	}

	return tokenutil.TokenFromIDAndSecret(
		string(tokenSecret.Data[tokenapi.BootstrapTokenIDKey]),
		string(tokenSecret.Data[tokenapi.BootstrapTokenSecretKey]),
	), nil
}

func getWorkloadClusterKubeClient(controllerClient client.Client) (client.Client, error) {
	var cluster *clusterv1.Cluster

	controlPlaneMachines, err := getControlPlaneMachines(controllerClient)
	if err != nil {
		return nil, err
	}

	if len(controlPlaneMachines) == 0 {
		return nil, fmt.Errorf("could not find control plane nodes")
	}

	clusterList := &clusterv1.ClusterList{}
	listOptions := &client.ListOptions{}
	err = controllerClient.List(context.TODO(), listOptions, clusterList)
	if err != nil {
		return nil, fmt.Errorf("error retrieving clusters: %v", err)
	}
	for _, c := range clusterList.Items {
		if clusterName, ok := controlPlaneMachines[0].Labels[clusterv1.MachineClusterLabelName]; ok && clusterName == c.Name {
			cluster = &c
		}
	}
	if cluster == nil {
		return nil, fmt.Errorf("could not find cluster")
	}

	kubeConfig, err := deployer.New().GetKubeConfig(cluster, controlPlaneMachines[0])
	if err != nil {
		return nil, err
	}

	apiConfig, err := clientcmd.Load([]byte(kubeConfig))
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.NewDefaultClientConfig(*apiConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, err
	}

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		return nil, fmt.Errorf("unable to create manager for restConfig: %v", err)
	}

	return mgr.GetClient(), nil
}

func startupScript(cluster *clusterv1.Cluster, machine *clusterv1.Machine, machineProviderSpec *providerv1.OpenstackProviderSpec, userdata, bootstrapToken string) (string, error) {
	clusterProviderSpec, err := providerv1.ClusterSpecFromProviderSpec(cluster.Spec.ProviderSpec)
	if err != nil {
		return "", err
	}

	if err := validateCertificates(clusterProviderSpec); err != nil {
		return "", err
	}

	kubeadmConfig, err := generateKubeadmConfig(util.IsControlPlaneMachine(machine), bootstrapToken, cluster, machine, machineProviderSpec, clusterProviderSpec)
	if err != nil {
		return "", err
	}

	params := setupParams{
		CACert:           string(clusterProviderSpec.CAKeyPair.Cert),
		CAKey:            string(clusterProviderSpec.CAKeyPair.Key),
		EtcdCACert:       string(clusterProviderSpec.EtcdCAKeyPair.Cert),
		EtcdCAKey:        string(clusterProviderSpec.EtcdCAKeyPair.Key),
		FrontProxyCACert: string(clusterProviderSpec.FrontProxyCAKeyPair.Cert),
		FrontProxyCAKey:  string(clusterProviderSpec.FrontProxyCAKeyPair.Key),
		SaCert:           string(clusterProviderSpec.SAKeyPair.Cert),
		SaKey:            string(clusterProviderSpec.SAKeyPair.Key),
		Machine:          machine,
		KubeadmConfig:    kubeadmConfig,
	}

	fMap := map[string]interface{}{
		"EscapeNewLines": templateEscapeNewLines,
		"Indent":         templateYAMLIndent,
	}
	startUpScript := template.Must(template.New("startUp").Funcs(fMap).Parse(userdata))

	var buf bytes.Buffer
	if err := startUpScript.Execute(&buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func validateCertificates(clusterProviderSpec *v1alpha1.OpenstackClusterProviderSpec) error {
	if !isKeyPairValid(clusterProviderSpec.CAKeyPair.Cert, clusterProviderSpec.CAKeyPair.Key) {
		return errors.New("CA cert material in the ClusterProviderSpec is missing cert/key")
	}

	if !isKeyPairValid(clusterProviderSpec.EtcdCAKeyPair.Cert, clusterProviderSpec.EtcdCAKeyPair.Key) {
		return errors.New("ETCD CA cert material in the ClusterProviderSpec is  missing cert/key")
	}

	if !isKeyPairValid(clusterProviderSpec.FrontProxyCAKeyPair.Cert, clusterProviderSpec.FrontProxyCAKeyPair.Key) {
		return errors.New("FrontProxy CA cert material in ClusterProviderSpec is  missing cert/key")
	}

	if !isKeyPairValid(clusterProviderSpec.SAKeyPair.Cert, clusterProviderSpec.SAKeyPair.Key) {
		return errors.New("ServiceAccount cert material in ClusterProviderSpec is  missing cert/key")
	}
	return nil
}

func isKeyPairValid(cert, key []byte) bool {
	return len(cert) > 0 && len(key) > 0
}

func templateEscapeNewLines(s string) string {
	return strings.ReplaceAll(s, "\n", "\\n")
}

func templateYAMLIndent(i int, input string) string {
	split := strings.Split(input, "\n")
	ident := "\n" + strings.Repeat(" ", i)
	return strings.Repeat(" ", i) + strings.Join(split, ident)
}
