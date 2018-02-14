# Go Chat

Golang chat app built mainly for learning Go.

## Dependencies

- [Docker](https://www.docker.com/)
- [Docker-Compose](https://docs.docker.com/compose/)
- [Glide](https://github.com/Masterminds/glide)

## Run

    ```bash
    glide up
    docker-compose up
    ```

## Redis Store

This utilizes [Redis](https://redis.io/) as a store for connection and user info.

## Session Management and Middleware

This uses [Gin](https://github.com/gin-gonic/gin) and [Gin Sessions](https://github.com/gin-contrib/sessions) for session management, tied into the Redis store, and for middleware.

## Google Integration

This uses [Google OAuth APIs](https://developers.google.com/identity/protocols/OAuth2) to allow a user to login through Google
so that go-chat can access their basic userinfo.

You will have to create a Google application through the [Google Developer Console](https://console.developers.google.com). You'll have to create a
Client ID and Client Secret for your application once it's created.

You will need to create a `creds.json` file that has the properties `cid` and `csecret`. These should contain the values for your Client ID and Client
Secret for your Google application. This will allow go-chat to use the Google APIs to get user data.

## Credit

- [Go Chat App](https://scotch.io/bar-talk/build-a-realtime-chat-server-with-go-and-websockets) for Gorilla WebSocket info.
- [Ramblings of a Build Engineer](https://skarlso.github.io/2016/06/12/google-signin-with-go/) was used as a guide for this.
