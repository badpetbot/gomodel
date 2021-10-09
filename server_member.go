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

// ServerMemberClientName is the name of the MgoDriver to use for ServerMember.
const ServerMemberClientName = "main"

// ServerMemberDBName is the name of the database to use for ServerMember.
const ServerMemberDBName = "badpetbot"

// ServerMemberColName is the name of the collection to use for ServerMember.
const ServerMemberColName = "server_members"

// ServerMemberCol gets a collection reference for ServerMember.
func ServerMemberCol() *mgo.Collection {
  return net.MgoCol(ServerMemberClientName, ServerMemberDBName, ServerMemberColName)
}

// INDICES:
// { _id: 1 }
// { discord_user_id: 1 }
// { discord_server_id: 1 }
// { discord_member_id: 1 }

// ServerMember is a single Discord "guild member". ServerMembers can belong to the same Discord "user" account,
// but for the purposes of BadPetBot, are considered separate users except for bans.
type ServerMember struct {
  // ID is a BSON ID generated in Create.
  ID                  bson.ObjectId   `bson:"_id"                   json:"_id"                    validate:"required"`
  DiscordUserID       string          `bson:"discord_user_id"       json:"discord_user_id"        validate:"required"`
  DiscordServerID     string          `bson:"discord_server_id"     json:"discord_server_id"      validate:"required"`
  DiscordMemberID     string          `bson:"discord_member_id"     json:"discord_member_id"      validate:"required"`
  CreatedAt           time.Time       `bson:"created_at"            json:"created_at"             validate:"required"`
  UpdatedAt           time.Time       `bson:"updated_at"            json:"updated_at"             validate:"required"`

  // Ownership relationships
  OwnerDiscordID      string          `bson:"owner_discord_id"      json:"owner_discord_id"       validate:"-"`
  SecOwnerDiscordIDs  []string        `bson:"sec_owner_discord_ids" json:"sec_owner_discord_ids"  validate:"-"`

  // Embeddables

  Owner               *ServerMember   `bson:"owner,omitalways"      json:"owner"                  validate:"-"`
  SecOwners           []ServerMember  `bson:"sec_owners,omitalways" json:"sec_owners"             validate:"-"`
}

// Create persists the document in the database. It can optionally run validations if present and
// prevent model persistence if they do not pass.
func (this *ServerMember) Create() error {

  // Ensure ID, timestamps, and tokens.
  this.ID = bson.NewObjectId()
  now := time.Now()
  this.CreatedAt = now
  this.UpdatedAt = now

  // Ensure defaults.

  // Run validations and return if they fail.
  if err := this.Validate(); err != nil {
    return err
  }

  // Persist the ServerMember.
  return net.MgoCol(ServerMemberClientName, ServerMemberDBName, ServerMemberColName).Insert(this)
}

// Update updates the document in the database. Important note, this function does NOT prepend
// the provided updates with "$set" or any other operator.
func (this *ServerMember) Update(updates bson.M) error {

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
  return net.MgoCol(ServerMemberClientName, ServerMemberDBName, ServerMemberColName).UpdateId(this.ID, updates)
}

// Delete permanently removes the document from the database.
func (this *ServerMember) Delete() error {

  // Delete the Link.
  return net.MgoCol(ServerMemberClientName, ServerMemberDBName, ServerMemberColName).RemoveId(this.ID)
}

// Validate runs validations against the model's fields.
func (this *ServerMember) Validate() error {

  // Implement validation rules here.
  return validation.NewValidator().Struct(this)
}

// CacheGetServerMember attempts to find a ServerMember by the key and value specified in cache before looking
// in the database and setting cache if found. If "negCache" is true, will check for neg-cache
// first, and also set neg-cache if the document wasn't found in the database either.
func CacheGetServerMember(key, value string, negCache bool) (*ServerMember, error) {

  client := net.RedisGetClient(ServerMemberClientName)
  cacheKey := ServerMemberClientName+":"+ServerMemberDBName+":"+ServerMemberColName+":"+key+":"+value

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
    server := new(ServerMember)
    err = json.Unmarshal([]byte(result), server)
    return server, err
  }

  // Get what's in the database.
  server := new(ServerMember)
  err := net.MgoCol(ServerMemberClientName, ServerMemberDBName, ServerMemberColName).Find(bson.M{
    key: value,
  }).One(server)

  // If it wasn't found and negCache is true, fill neg cache.
  if err == mgo.ErrNotFound && negCache {
    go fillNegCacheServerMember(client, cacheKey)

  // Else if there's no error, fill cache.
  } else if err != nil {
    go fillCacheServerMember(client, cacheKey, server)
  }
  return server, err
}

func fillCacheServerMember(client *redis.Client, key string, value *ServerMember) {
  serialized, err := json.Marshal(value)
  if err != nil {
    log.Warn().AnErr("fillCache", err).Msgf("Error serializing cache for ServerMember")
  }
  if err := client.Set(key, string(serialized), CacheTTL).Err(); err != nil {
    log.Warn().AnErr("fillCache", err).Msgf("Error filling cache for ServerMember")
  }
}

func fillNegCacheServerMember(client *redis.Client, key string) {
  if err := client.Set("neg:"+key, "neg", NegCacheTTL).Err(); err != nil {
    log.Warn().AnErr("fillNegCache", err).Msgf("Error filling neg cache for ServerMember")
  }
}

// Misc functions.