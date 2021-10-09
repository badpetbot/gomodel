/*
MODEL TEMPLATE

How to use:

1. Copy the file.
2. Replace (case sensitive) "ModelTemplate" with your model's ProperName.
3. Replace (case sensitive) "model_template" with your model's underscored_name.
4. Modify your fields and relationships.
5. Add your validations as needed. https://github.com/go-playground/validator
6. Comment your indices for easy reference later.
7. Change the comments!

FYI: Embeddable related documents only works because of the go.mod replacement
from globalsign/mgo to Nifty255/mgo, allowing the use of "omitalways" tags.

*/

package gomodel

import (

  // Import builtin packages.
  "encoding/json"
  "time"

  // Import 3rd party packages.
  "github.com/globalsign/mgo"
  "github.com/globalsign/mgo/bson"
  "github.com/go-redis/redis"
  "github.com/rs/zerolog/log"

  // Import internal packages.
  "github.com/badpetbot/gocommon/net"
  "github.com/badpetbot/gocommon/validation"
)

// ModelTemplateClientName is the name of the MgoDriver to use for ModelTemplate.
const ModelTemplateClientName = "main"

// ModelTemplateDBName is the name of the database to use for ModelTemplate.
const ModelTemplateDBName = "badpetbot"

// ModelTemplateColName is the name of the collection to use for ModelTemplate.
const ModelTemplateColName = "model_templates"

// ModelTemplateCol gets a collection reference for ModelTemplate.
func ModelTemplateCol() *mgo.Collection {
  return net.MgoCol(ModelTemplateClientName, ModelTemplateDBName, ModelTemplateColName)
}

// INDICES:
// { _id: 1 }

// ModelTemplate is a model template, meant to be copied.
type ModelTemplate struct {
  // ID is a BSON ID generated in Create.
  ID                  bson.ObjectId   `bson:"_id"                           json:"_id"                  validate:"required"`
  CreatedAt           time.Time       `bson:"created_at"                    json:"created_at"           validate:"required"`
  UpdatedAt           time.Time       `bson:"updated_at"                    json:"updated_at"           validate:"required"`
  FieldWithDefault    int             `bson:"field_with_default"            json:"field_with_default"   validate:"gt=2,lt=10"`

  // Relationship IDs. Referencing another document's ID causes this document to "belong to" that document. A document can
  // belong to one other document, or many. A document to which this one blongs can "have" many other documents of this type.
  // For a generic example, a "session" may belong to 1 user, and each user may have many sessions.
  RelatedTemplateID   *bson.ObjectId  `bson:"related_template_id"           json:"related_template_id"  validate:"-"`
  RelatedTemplateIDs  []bson.ObjectId `bson:"related_template_ids"          json:"related_template_ids" validate:"-"`

  // Embeddables. Can be pulled in via aggregations, "omitalways" on the bson tag prevents storing embedded documents
  // which are meant only to be related objects. Embeddable references can be declared without "belonging" IDs. This sort
  // of relationship is known as "has" one/many. The child model is responsible for "belonging" to this one in that case.
  RelatedTemplate     *ModelTemplate  `bson:"related_template,omitalways"   json:"related_template"     validate:"-"`
  RelatedTemplates    []ModelTemplate `bson:"related_templates,omitalways"  json:"related_templates"    validate:"-"`
}

// Create persists the document in the database. It can optionally run validations if present and
// prevent model persistence if they do not pass.
func (this *ModelTemplate) Create() error {

  // Ensure ID, timestamps, and tokens.
  this.ID = bson.NewObjectId()
  now := time.Now()
  this.CreatedAt = now
  this.UpdatedAt = now

  // Ensure defaults.
  this.FieldWithDefault = 7

  // Run validations and return if they fail.
  if err := this.Validate(); err != nil {
    return err
  }

  // Persist the ModelTemplate.
  return net.MgoCol(ModelTemplateClientName, ModelTemplateDBName, ModelTemplateColName).Insert(this)
}

// Update updates the document in the database. Important note, this function does NOT prepend
// the provided updates with "$set" or any other operator.
func (this *ModelTemplate) Update(updates bson.M) error {

  // Update updated-at timestamp.
  this.UpdatedAt = time.Now()
  _, setting := updates["$set"]
  if !setting {
    updates["$set"] = bson.M{}
  }
  updates["$set"].(bson.M)["updated_at"] = this.UpdatedAt

  if err := this.Validate(); err != nil {
    return err
  }

  // Persist the updates.
  return net.MgoCol(ModelTemplateClientName, ModelTemplateDBName, ModelTemplateColName).UpdateId(this.ID, updates)
}

// Delete permanently removes the document from the database.
func (this *ModelTemplate) Delete() error {

  // Delete the Link.
  return net.MgoCol(ModelTemplateClientName, ModelTemplateDBName, ModelTemplateColName).RemoveId(this.ID)
}

// Validate runs validations against the model's fields.
func (this *ModelTemplate) Validate() error {

  // Implement validation rules here.
  return validation.NewValidator().Struct(this)
}

// Cache functions.

// CacheGetModelTemplate attempts to find a ModelTemplate by the key and value specified in cache before looking
// in the database and setting cache if found. If "negCache" is true, will check for neg-cache
// first, and also set neg-cache if the document wasn't found in the database either.
func CacheGetModelTemplate(key, value string, negCache bool) (*ModelTemplate, error) {

  client := net.RedisGetClient(ModelTemplateClientName)
  cacheKey := ModelTemplateClientName+":"+ModelTemplateDBName+":"+ModelTemplateColName+":"+key+":"+value

  // Return not-found early if neg-cache exists.
  if negCache {
    if result, err := client.Get("neg:"+cacheKey).Result(); err != nil {
      return nil, err
    } else if result != "" {
      return nil, mgo.ErrNotFound
    }
  }

  // Return what's in cache if it's found.
  if result, err := client.Get(cacheKey).Result(); err != nil {
    return nil, err
  } else if result != "" {
    server := new(ModelTemplate)
    err = json.Unmarshal([]byte(result), server)
    return server, err
  }

  // Get what's in the database.
  server := new(ModelTemplate)
  err := net.MgoCol(ModelTemplateClientName, ModelTemplateDBName, ModelTemplateColName).Find(bson.M{
    key: value,
  }).One(server)

  // If it wasn't found and negCache is true, fill neg cache.
  if err == mgo.ErrNotFound && negCache {
    go fillNegCacheModelTemplate(client, cacheKey)

  // Else if there's no error, fill cache.
  } else if err != nil {
    go fillCacheModelTemplate(client, cacheKey, server)
  }
  return server, err
}

func fillCacheModelTemplate(client *redis.Client, key string, value *ModelTemplate) {
  serialized, err := json.Marshal(value)
  if err != nil {
    log.Warn().AnErr("fillCache", err).Msgf("Error serializing cache for ModelTemplate")
  }
  if err := client.Set(key, string(serialized), CacheTTL).Err(); err != nil {
    log.Warn().AnErr("fillCache", err).Msgf("Error filling cache for ModelTemplate")
  }
}

func fillNegCacheModelTemplate(client *redis.Client, key string) {
  if err := client.Set("neg:"+key, "neg", NegCacheTTL).Err(); err != nil {
    log.Warn().AnErr("fillNegCache", err).Msgf("Error filling neg cache for ModelTemplate")
  }
}

// Misc functions.