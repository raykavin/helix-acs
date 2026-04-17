package device

import (
	"context"
	"fmt"

	"github.com/raykavin/helix-acs/packages/logger"
)

type service struct {
	repo   Repository
	logger logger.Logger
}

// NewService creates a new Service backed by the given Repository.
func NewService(repo Repository, logger logger.Logger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

// UpsertFromInform creates or updates a device record from a TR-069 Inform message.
func (s *service) UpsertFromInform(ctx context.Context, req *UpsertRequest) (*Device, error) {
	s.logger.
		WithField("serial", req.Serial).
		WithField("manufacturer", req.Manufacturer).
		WithField("model", req.ModelName).
		Debug("Upserting device from inform")

	device, err := s.repo.Upsert(ctx, req)
	if err != nil {
		s.logger.
			WithError(err).
			WithField("serial", req.Serial).
			Error("Failed to upsert device")

		return nil, fmt.Errorf("upsert device %s: %w", req.Serial, err)
	}

	s.logger.
		WithField("serial", device.Serial).
		WithField("id", device.ID.Hex()).
		Info("Device upserted successfully")

	return device, nil
}

// FindBySerial retrieves a device by its serial number.
func (s *service) FindBySerial(ctx context.Context, serial string) (*Device, error) {
	s.logger.WithField("serial", serial).
		Debug("Finding device by serial")

	device, err := s.repo.FindBySerial(ctx, serial)
	if err != nil {
		s.logger.WithError(err).
			WithField("serial", serial).
			Error("Failed to find device")

		return nil, fmt.Errorf("find device %s: %w", serial, err)
	}
	return device, nil
}

// List returns a paginated list of devices matching the given filter.
func (s *service) List(ctx context.Context, filter DeviceFilter, page, limit int) ([]*Device, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	skip := int64((page - 1) * limit)

	s.logger.
		WithField("page", page).
		WithField("limit", limit).
		Debug("Listing devices")

	devices, total, err := s.repo.Find(ctx, filter, skip, int64(limit))
	if err != nil {
		s.logger.WithError(err).Error("Failed to list devices")
		return nil, 0, fmt.Errorf("list devices: %w", err)
	}

	s.logger.
		WithField("total", total).
		WithField("returned", len(devices)).
		Debug("Devices listed")

	return devices, total, nil
}

// UpdateTags replaces the tags on a device and returns the updated record.
func (s *service) UpdateTags(ctx context.Context, serial string, tags []string) (*Device, error) {
	s.logger.
		WithField("serial", serial).
		WithField("tags", tags).
		Debug("Updating device tags")

	if err := s.repo.UpdateTags(ctx, serial, tags); err != nil {
		s.logger.WithError(err).
			WithField("serial", serial).
			Error("Failed to update tags")
		return nil, fmt.Errorf("update tags for device %s: %w", serial, err)
	}

	device, err := s.repo.FindBySerial(ctx, serial)
	if err != nil {
		return nil, fmt.Errorf("fetch device after tag update %s: %w", serial, err)
	}

	s.logger.WithField("serial", serial).Debug("Device tags updated")
	return device, nil
}

// Delete removes a device by its serial number.
func (s *service) Delete(ctx context.Context, serial string) error {
	s.logger.WithField("serial", serial).Debug("Deleting device")

	if err := s.repo.Delete(ctx, serial); err != nil {
		s.logger.
			WithError(err).
			WithField("serial", serial).
			Error("Failed to delete device")
		return fmt.Errorf("delete device %s: %w", serial, err)
	}

	s.logger.WithField("serial", serial).Debug("Device deleted")
	return nil
}

// UpdateInfo merges rich sub-documents into the device record.
func (s *service) UpdateInfo(ctx context.Context, serial string, upd InfoUpdate) error {
	if err := s.repo.UpdateInfo(ctx, serial, upd); err != nil {
		s.logger.WithError(err).WithField("serial", serial).Debug("Failed to update device info")
		return fmt.Errorf("update device info %s: %w", serial, err)
	}
	return nil
}

// SetOnline updates the online presence flag for a device.
func (s *service) SetOnline(ctx context.Context, serial string, online bool) error {
	s.logger.
		WithField("serial", serial).
		WithField("online", online).
		Debug("Setting device online status")

	if err := s.repo.SetOnline(ctx, serial, online); err != nil {
		s.logger.
			WithError(err).
			WithField("serial", serial).
			Error("Failed to set online status")
		return fmt.Errorf("set online for device %s: %w", serial, err)
	}

	s.logger.
		WithField("serial", serial).
		WithField("online", online).
		Debug("Device online status updated")

	return nil
}
