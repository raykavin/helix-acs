package device

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionName = "devices"

type mongoRepository struct {
	col *mongo.Collection
}

// NewMongoRepository creates a new MongoDB-backed Repository and ensures
// indexes are created before returning.
func NewMongoRepository(ctx context.Context, db *mongo.Database) (Repository, error) {
	col := db.Collection(collectionName)
	r := &mongoRepository{col: col}
	if err := r.createIndexes(ctx); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *mongoRepository) createIndexes(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "serial", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("serial_unique"),
		},
		{
			Keys:    bson.D{{Key: "online", Value: 1}, {Key: "last_inform", Value: -1}},
			Options: options.Index().SetName("online_last_inform"),
		},
		{
			Keys:    bson.D{{Key: "wan_ip", Value: 1}},
			Options: options.Index().SetName("wan_ip"),
		},
	}

	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	return err
}

// Upsert inserts or updates a device identified by its serial number.
func (r *mongoRepository) Upsert(ctx context.Context, req *UpsertRequest) (*Device, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	now := time.Now().UTC()

	filter := bson.M{"serial": req.Serial}

	setFields := bson.M{
		"serial":        req.Serial,
		"oui":           req.OUI,
		"manufacturer":  req.Manufacturer,
		"model_name":    req.ModelName,
		"product_class": req.ProductClass,
		"data_model":    req.DataModel,
		"schema":        req.Schema,
		"ip_address":    req.IPAddress,
		"wan_ip":        req.WANIP,
		"sw_version":    req.SWVersion,
		"hw_version":    req.HWVersion,
		"bl_version":    req.BLVersion,
		"parameters":    req.Parameters,
		"online":        true,
		"last_inform":   now,
		"updated_at":    now,
	}
	// Only overwrite system fields when the CPE reports them.
	if req.UptimeSeconds > 0 {
		setFields["uptime_seconds"] = req.UptimeSeconds
	}
	if req.RAMTotal > 0 {
		setFields["ram_total"] = req.RAMTotal
	}
	if req.RAMFree > 0 {
		setFields["ram_free"] = req.RAMFree
	}
	if req.ACSURL != "" {
		setFields["acs_url"] = req.ACSURL
	}

	update := bson.M{
		"$set": setFields,
		"$setOnInsert": bson.M{
			"created_at": now,
			"tags":       []string{},
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var device Device
	if err := r.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&device); err != nil {
		return nil, err
	}
	return &device, nil
}

// UpdateInfo merges rich sub-documents (WAN, LAN, WiFi, connected hosts) into
// the device document without touching other fields.
func (r *mongoRepository) UpdateInfo(ctx context.Context, serial string, upd InfoUpdate) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	setFields := bson.M{"updated_at": time.Now().UTC()}

	if upd.WAN != nil {
		setFields["wan"] = upd.WAN
	}
	if upd.LAN != nil {
		setFields["lan"] = upd.LAN
	}
	if upd.WiFi24 != nil {
		setFields["wifi_24"] = upd.WiFi24
	}
	if upd.WiFi5 != nil {
		setFields["wifi_5"] = upd.WiFi5
	}
	if upd.ConnectedHosts != nil {
		setFields["connected_hosts"] = upd.ConnectedHosts
	}

	if len(setFields) == 1 {
		return nil // nothing to update
	}

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"serial": serial},
		bson.M{"$set": setFields},
	)
	return err
}

// FindBySerial returns a device by its serial number.
func (r *mongoRepository) FindBySerial(ctx context.Context, serial string) (*Device, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var device Device
	if err := r.col.FindOne(ctx, bson.M{"serial": serial}).Decode(&device); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &device, nil
}

// Find returns a paginated list of devices matching the given filter.
func (r *mongoRepository) Find(ctx context.Context, filter DeviceFilter, skip, limit int64) ([]*Device, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	query := bson.M{}

	if filter.Online != nil {
		query["online"] = *filter.Online
	}
	if filter.Manufacturer != "" {
		query["manufacturer"] = filter.Manufacturer
	}
	if filter.ModelName != "" {
		query["model_name"] = filter.ModelName
	}
	if filter.Tag != "" {
		query["tags"] = filter.Tag
	}
	if filter.WANIP != "" {
		query["wan_ip"] = filter.WANIP
	}
	if filter.Serial != "" {
		query["serial"] = filter.Serial
	}

	total, err := r.col.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.D{{Key: "last_inform", Value: -1}})

	cursor, err := r.col.Find(ctx, query, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var devices []*Device
	if err := cursor.All(ctx, &devices); err != nil {
		return nil, 0, err
	}
	return devices, total, nil
}

// UpdateTags replaces the tags array for the given device.
func (r *mongoRepository) UpdateTags(ctx context.Context, serial string, tags []string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"serial": serial},
		bson.M{"$set": bson.M{"tags": tags, "updated_at": time.Now().UTC()}},
	)
	return err
}

// Delete removes a device by serial number.
func (r *mongoRepository) Delete(ctx context.Context, serial string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := r.col.DeleteOne(ctx, bson.M{"serial": serial})
	return err
}

// SetOnline updates the online status of a device.
func (r *mongoRepository) SetOnline(ctx context.Context, serial string, online bool) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"serial": serial},
		bson.M{"$set": bson.M{"online": online, "updated_at": time.Now().UTC()}},
	)
	return err
}

// UpdateParameters merges the given parameters map into the device's stored parameters.
func (r *mongoRepository) UpdateParameters(ctx context.Context, serial string, params map[string]string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	setFields := bson.M{"updated_at": time.Now().UTC()}
	for k, v := range params {
		setFields["parameters."+k] = v
	}

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"serial": serial},
		bson.M{"$set": setFields},
	)
	return err
}
