module github.com/ruslanBik4/dbEngine

go 1.14

require (
	github.com/jackc/pgconn v1.6.1
	github.com/jackc/pgproto3/v2 v2.0.2
	github.com/jackc/pgx/v4 v4.7.1
	github.com/pkg/errors v0.9.1
	github.com/ruslanBik4/httpgo v1.1.10013
	github.com/stretchr/testify v1.6.1
	golang.org/x/net v0.0.0-20200625001655-4c5254603344
)


replace bitbucket.org/PinIdea/fcgi_client => ../httpgo/models/fcgi_client
