package gomodel

import (

  // Import builtin packages.
  "time"

  // Import 3rd party packages.
  "github.com/globalsign/mgo"
  "github.com/globalsign/mgo/bson"

  // Import internal packages.
  "github.com/badpetbot/gocommon/net"
  "github.com/badpetbot/gocommon/validation"
)

// ServerClientName is the name of the MgoDriver to use for Server.
const ServerClientName = "main"

// ServerDBName is the name of the database to use for Server.
const ServerDBName = "badpetbot"

// ServerColName is the name of the collection to use for Server.
const ServerColName = "servers"

// ServerCol gets a collection reference for Server.
func ServerCol() *mgo.Collection {
  return net.MgoCol(ServerClientName, ServerDBName, ServerColName)
}

// INDICES:
// { _id: 1 }
// { discord_id: 1 }

// Server is a single Discord "guild" (colloquially known as a server).
type Server struct {
  // ID is a BSON ID generated in Create.
  ID        bson.ObjectId   `bson:"_id"         json:"_id"        validate:"required"`
  DiscordID string          `bson:"discord_id"  json:"discord_id" validate:"required"`
  CreatedAt time.Time       `bson:"created_at"  json:"created_at" validate:"required"`
  UpdatedAt time.Time       `bson:"updated_at"  json:"updated_at" validate:"required"`
}

// Create persists the document in the database. It can optionally run validations if present and
// prevent model persistence if they do not pass.
func (this *Server) Create() error {

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

  // Persist the Server.
  return net.MgoCol(ServerClientName, ServerDBName, ServerColName).Insert(this)
}

// Update updates the document in the database. Important note, this function does NOT prepend
// the provided updates with "$set" or any other operator.
func (this *Server) Update(updates bson.M) error {

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
  return net.MgoCol(ServerClientName, ServerDBName, ServerColName).UpdateId(this.ID, updates)
}

// Delete permanently removes the document from the database.
func (this *Server) Delete() error {

  // Delete the Link.
  return net.MgoCol(ServerClientName, ServerDBName, ServerColName).RemoveId(this.ID)
}

// Validate runs validations against the model's fields.
func (this *Server) Validate() error {

  // Implement validation rules here.
  return validation.NewValidator().Struct(this)
}

// Misc functions.