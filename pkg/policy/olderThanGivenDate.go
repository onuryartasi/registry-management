package policy

import (
	"encoding/json"
	"github.com/onuryartasi/registry-cleaner/pkg/registry"
	"log"
	"time"
)

func (policy Policy) olderThanGivenDateCheck(image registry.Image) registry.Image {


	parsedDate, err := parseDate(policy.OlderThanGivenDateRule.Date)
	if err != nil {
		log.Println("Cannot parse given date with static time layout. Check layout table...")
		return image
	}
	var tagList []registry.Tag
	var v1Compatibility registry.V1Compatibility
	var deletableTags []string

	for _, tag := range image.Tags {

		manifests := client.GetManifest(image.Name, tag)
		//v1comp,err := strconv.Unquote(manifests.History[0].V1Compatibility)
		if len(manifests.History) == 0 {
			log.Println("Image Manifest is broken.Skipping this tag.", image.Name, tag)
			continue
		}

		v1comp := manifests.History[0].V1Compatibility
		err := json.Unmarshal([]byte(v1comp), &v1Compatibility)
		if err != nil {
			log.Println("Error Unmarshal compatibility ", err)
		}

		digest := client.GetDigest(image.Name, tag)
		tagList = append(tagList, registry.Tag{Name: tag, CreatedDate: v1Compatibility.Created, Digest: digest, ImageName: image.Name})


		if parsedDate.After(v1Compatibility.Created) {
			deletableTags = append(deletableTags, tag)
		}
		//log.Printf("Image %s, Tag: %s, created date: %v", image.Name ,tag, startedTime.Sub(v1Compatibility.Created).Hours())
	}
	log.Println("Deletable images: ", deletableTags)
	return registry.Image{Name: image.Name, Tags: deletableTags}

}

func parseDate(date string) (time.Time, error) {
	var err error
	var parsedDate time.Time
	var layouts = []string{
		"02.01.2006 15:04:05",
		"02.01.2006 15:04",
		"02.01.2006",
	}

	for _, layout := range layouts {
		parsedDate, err = time.Parse(layout, date)
		if err != nil {
			continue
		}
		return parsedDate, err
	}
	return time.Time{}, err
}