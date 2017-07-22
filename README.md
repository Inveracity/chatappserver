# React chatapp

To run the code you need RethinkDB and the frontend site written in ReactJS which can be found here: [chatappfrontend](https://github.com/Inveracity/chatappfrontend.git)

```bash
git clone https://github.com/Inveracity/chatappserver.git

go get -u github.com/mitchellh/mapstructure
go get -u github.com/gorilla/websocket
go get -u gopkg.in/gorethink/gorethink.v3

cd chatappserver

# Windows
go run client.go handler.go main.go router.go

# not Windows
go run *.go

```

http://localhost:8080


# Acknowledgement:

Original code comes from James Moore and his course in _ReactJS, Go and RethinkDB for realtime webapp development_