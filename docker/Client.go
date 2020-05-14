package docker

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func initClient() *client.Client {

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	return dockerClient
}

type NetworkList map[string]types.NetworkResource
type NetworkItems map[string]types.EndpointResource

type ServiceList map[string]network.ServiceInfo

type Response struct {
	Services ServiceList
	Networks NetworkList
}

func CatchEvents() (<-chan events.Message, <-chan error) {
	dockerClient := initClient()

	return dockerClient.Events(context.Background(), types.EventsOptions{})
}

func GetData() Response {
	dockerClient := initClient()
	res := Response{
		Services: ServiceList{},
		Networks: NetworkList{},
	}
	res.Networks = listNetworks(dockerClient)
	for _, net := range res.Networks {
		for serviceName, service := range net.Services {
			if _, ok := res.Services[serviceName]; !ok {
				res.Services[serviceName] = service
			}
		}
	}
	return res
}
func listNetworks(dockerClient *client.Client) map[string]types.NetworkResource {
	var networksToReturn = map[string]types.NetworkResource{}
	ctx := context.Background()
	networks, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}
	for _, networkItem := range networks {
		network, _ := dockerClient.NetworkInspect(ctx, networkItem.ID, types.NetworkInspectOptions{Verbose: true})
		networksToReturn[network.Name] = network
	}
	return networksToReturn
}

func GetContainerServiceName(containerId string) *string {
	client := initClient()
	container, _ := client.ContainerInspect(context.Background(), containerId)
	if serviceName, ok := container.Config.Labels["com.docker.swarm.service.name"]; ok {
		return &serviceName
	}
	return nil
}
