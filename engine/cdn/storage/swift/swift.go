package swift

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/ncw/swift"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type Swift struct {
	client swift.Connection
	storage.AbstractUnit
	encryption.ConvergentEncryption
	config storage.SwiftStorageConfiguration
}

var _ storage.StorageUnit = new(Swift)

func init() {
	storage.RegisterDriver("swift", new(Swift))
}

func (s *Swift) Init(cfg interface{}) error {
	config, is := cfg.(*storage.SwiftStorageConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.config = *config
	s.ConvergentEncryption = encryption.New(config.Encryption)
	s.client = swift.Connection{
		AuthUrl:  config.Address,
		Region:   config.Region,
		Tenant:   config.Tenant,
		Domain:   config.Domain,
		UserName: config.Username,
		ApiKey:   config.Password,
	}
	return nil
}

func (s *Swift) ItemExists(i index.Item) (bool, error) {
	iu, err := s.ExistsInDatabase(i.ID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	container, object, err := s.getItemPath(*iu)
	if err != nil {
		return false, err
	}

	allObjs, _ := s.client.ObjectNamesAll(container, nil)
	for i := range allObjs {
		if allObjs[i] == object {
			return true, nil
		}
	}
	return false, nil
}

func (s *Swift) NewWriter(i storage.ItemUnit) (io.WriteCloser, error) {
	container, object, err := s.getItemPath(i)
	if err != nil {
		return nil, err
	}

	if err := s.client.ContainerCreate(container, nil); err != nil {
		return nil, sdk.WrapError(err, "Unable to create container %s", container)
	}

	file, err := s.client.ObjectCreate(container, object, true, "", "application/octet-stream", nil)
	if err != nil {
		return nil, sdk.WrapError(err, "SwiftStore> Unable to create object %s", object)
	}

	return file, nil
}

func (s *Swift) NewReader(i storage.ItemUnit) (io.ReadCloser, error) {
	container, object, err := s.getItemPath(i)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	go func() {
		if _, err = s.client.ObjectGet(container, object, pw, true, nil); err != nil {
			log.Error(context.Background(), "unable to get object %s/%s: %v", container, object, err)
			return
		}
		if err := pw.Close(); err != nil {
			log.Error(context.Background(), "unable to close pipewriter %s/%s: %v", container, object, err)
			return
		}
	}()

	return pr, nil
}

func (s *Swift) getItemPath(i storage.ItemUnit) (container string, object string, err error) {
	loc := i.Locator
	container = fmt.Sprintf("%s-%s-%s", s.config.ContainerPrefix, i.Item.Type, loc[:3])
	object = loc
	container, object = escape(container, object)
	return container, object, nil
}

func escape(container, object string) (string, string) {
	container = url.QueryEscape(container)
	container = strings.Replace(container, "/", "-", -1)
	object = url.QueryEscape(object)
	object = strings.Replace(object, "/", "-", -1)
	return container, object
}
