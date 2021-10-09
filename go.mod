module github.com/badpetbot/gomodel

go 1.14

require (
	github.com/badpetbot/gocommon v0.0.0-20211009221702-8962210fd7eb
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/rs/zerolog v1.25.0
)

replace github.com/globalsign/mgo => github.com/Nifty255/mgo v0.0.0-20200423052436-ae3b558ebcf4
