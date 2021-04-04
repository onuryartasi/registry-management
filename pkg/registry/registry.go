package registry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/onuryartasi/registry-cleaner/pkg/logging"
)

var logger *logrus.Logger

func init() {
	logger = logging.GetLogger()
}

// NewClient return Registry object for reuse.
func NewClient(host, port string, dryRun bool) Registry {
	return Registry{HOST: host, PORT: port, DryRun: dryRun}
}

//BasicAuthentication Set basic auth given registry.
func (registry *Registry) BasicAuthentication(user, password string) {
	*registry = Registry{HOST: registry.HOST, PORT: registry.PORT, USER: user, PASSWORD: password, DryRun: registry.DryRun}
}

//GET return  http response for given path.
func (registry Registry) GET(path string) (*http.Response, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:%s%s", registry.HOST, registry.PORT, path))
	return resp, err
}

//getCatalog return v2 catalog for given registry.
func (registry Registry) GetCatalog() Catalog {
	var catalog Catalog
	resp, err := registry.GET("/v2/_catalog")
	if err != nil {
		logger.Fatalln("Error getting version", err)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Errorln(err)
	}

	err = json.Unmarshal(bodyBytes, &catalog)
	if err != nil {
		logger.Errorln("Unmarshall error ", err)
	}

	return catalog
}

//splitRepositories split group and image name ex. foo/bar:latest => group = foo, image = bar (Default group is empty string)
func SplitRepositories(repositories []string) map[string][]string {
	var group, repoName string
	var registryMap = make(map[string][]string)
	for _, repo := range repositories {
		splitted := strings.Split(repo, "/")
		if len(splitted) == 1 {
			group = ""
			repoName = splitted[0]
		} else {
			// TODO: refactor
			repoName = splitted[len(splitted)-1]
			group = ""
			subSplitted := splitted[0 : len(splitted)-1]
			for i, v := range subSplitted {
				if i == 0 {
					group = group + v
				} else {
					group = group + "/" + v
				}
			}
		}

		groupRepositories, ok := registryMap[group]
		if ok {
			groupRepositories = append(groupRepositories, repoName)
			registryMap[group] = groupRepositories
		} else {
			registryMap[group] = []string{repoName}
		}
	}

	return registryMap
}

//getDigest return image's digest with `application/vnd.docker.distribution.manifest.v2+json`
func (registry Registry) GetDigest(imageName, tag string) (string, error) {
	client := &http.Client{}
	var digest string

	url := fmt.Sprintf("http://%s:%s/v2/%s/manifests/%s", registry.HOST, registry.PORT, imageName, tag)
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Println("Cannot get docker image digest", err)
		return "", err
	}

	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Cannot get digest", err)
		return "", err
	}
	if resp.StatusCode == 200 {
		digest = resp.Header["Docker-Content-Digest"][0]
	} else {
		return "", fmt.Errorf("Cannot get digest statusCode:%v", resp.StatusCode)
	}
	return digest, nil
}

func (registry Registry) GetManifest(imageName, tag string) Manifests {
	var manifests Manifests
	url := fmt.Sprintf("http://%s:%s/v2/%s/manifests/%s", registry.HOST, registry.PORT, imageName, tag)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Cannot get docker image digest", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Cannot get digest", err)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(bodyBytes, &manifests)
	if err != nil {
		log.Println("Unmarshal error mamifes", err)
	}

	return manifests
}

func (registry Registry) GetImageTags(groupName, repoName string) Image {
	var image Image
	var url string
	if len(groupName) > 0 {
		url = fmt.Sprintf("http://%s:%s/v2/%s/%s/tags/list", registry.HOST, registry.PORT, groupName, repoName)
	} else {
		url = fmt.Sprintf("http://%s:%s/v2/%s/tags/list", registry.HOST, registry.PORT, repoName)
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(bodyBytes, &image)
	if err != nil {
		log.Println("Error unmarshal tags", err)
	}

	return image
}

func (registry Registry) DeleteTag(imageName, digest string) (int, error) {
	url := fmt.Sprintf("http://%s:%s/v2/%s/manifests/%s", registry.HOST, registry.PORT, imageName, digest)
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		logger.Errorln("Cannot construct new request", err)
		return 0, err
	}

	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	if len(registry.USER) > 0 {
		req.SetBasicAuth(registry.USER, registry.PASSWORD)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("Cannot delete tag with digest %s:%s error: %s", imageName, digest, err)
		return resp.StatusCode, err
	}

	return resp.StatusCode, nil
}
