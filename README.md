# OAuth authorization server

## Database migrations

Uses [migrate](https://github.com/mattes/migrate):

```
migrate -source "file://db/migration/" -database "mysql://tick:tick@tcp(localhost:3306)/tick" up 1
```