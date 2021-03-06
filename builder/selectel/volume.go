package selectel

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v2/volumes"
)

// StateRefreshFunc is a function type used for StateChangeConf that is
// responsible for refreshing the item being watched for a state change.
//
// It returns three results. `result` is any object that will be returned
// as the final object after waiting for state change. This allows you to
// return the final updated object, for example an openstack instance after
// refreshing it.
//
// `state` is the latest state of that object. And `err` is any error that
// may have happened while refreshing the state.

// VolumeV2StateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an OpenStack volume.
func VolumeV2StateRefreshFunc(
	client *gophercloud.ServiceClient, volumeID string) StateRefreshFunc {
	return func() (interface{}, string, int, error) {
		v, err := volumes.Get(client, volumeID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[INFO] 404 on ServerStateRefresh, returning DELETED")
				return v, "deleted", 0, nil
			}
			log.Printf("[ERROR] Error on ServerStateRefresh: %s", err)
			return nil, "", 0, err
		}

		if v.Status == "error" {
			return v, v.Status, 0, fmt.Errorf("There was an error creating the volume. " +
				"Please check with your cloud admin or check the Block Storage " +
				"API logs to see why this error occurred.")
		}

		return v, v.Status, 50, nil
	}
}
