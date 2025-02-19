package presenters

import (
	"gorm.io/datatypes"

	"github.com/openshift-online/maestro/pkg/api"
	"github.com/openshift-online/maestro/pkg/api/openapi"
	"github.com/openshift-online/maestro/pkg/constants"
	"github.com/openshift-online/maestro/pkg/util"
)

// ConvertResource converts a resource from the API to the openapi representation.
func ConvertResource(resource openapi.Resource) (*api.Resource, error) {
	payload, err := ConvertResourceManifest(resource.Manifest, resource.GroupResource, resource.DeleteOption, resource.UpdateStrategy)
	if err != nil {
		return nil, err
	}
	return &api.Resource{
		Name: util.NilToEmptyString(resource.Name),
		Meta: api.Meta{
			ID: util.NilToEmptyString(resource.Id),
		},
		ConsumerName: util.NilToEmptyString(resource.ConsumerName),
		Version:      util.NilToEmptyInt32(resource.Version),
		// Set the default source ID for RESTful API calls and do not allow modification
		Source:  constants.DefaultSourceID,
		Type:    api.ResourceTypeSingle,
		Payload: payload,
	}, nil
}

// ConvertResourceManifest converts a resource manifest from the openapi representation to the API.
func ConvertResourceManifest(manifest, groupResource, deleteOption, updateStrategy map[string]interface{}) (datatypes.JSONMap, error) {
	return api.EncodeManifest(manifest, groupResource, deleteOption, updateStrategy)
}

// PresentResource converts a resource from the API to the openapi representation.
func PresentResource(resource *api.Resource) (*openapi.Resource, error) {
	manifest, groupResource, deleteOption, updateStrategy, err := api.DecodeManifest(resource.Payload)
	if err != nil {
		return nil, err
	}
	status, err := api.DecodeStatus(resource.Status)
	if err != nil {
		return nil, err
	}
	reference := PresentReference(resource.ID, resource)
	res := &openapi.Resource{
		Id:             reference.Id,
		Kind:           reference.Kind,
		Href:           reference.Href,
		Name:           openapi.PtrString(resource.Name),
		ConsumerName:   openapi.PtrString(resource.ConsumerName),
		Version:        openapi.PtrInt32(resource.Version),
		CreatedAt:      openapi.PtrTime(resource.CreatedAt),
		UpdatedAt:      openapi.PtrTime(resource.UpdatedAt),
		Manifest:       manifest,
		GroupResource:  groupResource,
		DeleteOption:   deleteOption,
		UpdateStrategy: updateStrategy,
		Status:         status,
	}

	// set the deletedAt field if the resource has been marked as deleted
	if !resource.DeletedAt.Time.IsZero() {
		res.DeletedAt = openapi.PtrTime(resource.DeletedAt.Time)
	}

	return res, nil
}
