# Backend Challenge

A Simple Web Chat API

## Requirements

* [Docker](https://docs.docker.com/)
* [Docker Compose](https://docs.docker.com/compose/)

## Development

You can run the app with:

```shell
docker-compose up
```

Then, you should be able to access it at [127.0.0.1:8080](http://127.0.0.1:8080).

## Build

Build with:

```shell
docker build -t backend-challenge .
```

And run with:

```shell
docker run --name backend-challenge -d --publish 8080:8080 \
  --env DB_CONNSTR='postgres://<user>@<host>/<dbname>?sslmode=disable'
  backend-challenge
```

## Test

You can run the test suite with:

```shell
docker-compose -f docker-compose.test.yml -p backend-challenge_test up sut
```

---

# Ada Support Backend Coding Challenge

Hello there! :wave:

This is our challenge for potential new backend & infrastructure developer team members. We'd like to see how you tackle an open-ended project in Ada's domain (chat). We don't mind what language you use to complete the challenge. Feel free to try something new, or to use technology that you're already comfortable with. We're partial to Python here at Ada, but please feel free to use whatever you'd like :smile:

## Your Quest

We'd like you to design and build a simple web service responsible for two things:

1. Accept incoming chat messages over HTTP
2. Serve up conversation history over HTTP

When you're done, please open a pull request in this repository and we'll take a look! Expect us to give some feedback and ask questions to better understand your thought process.

### Important Notes

- Please don't spend too long on this! It shouldn't take more than 3 hours with a familiar technology stack.
- Please reach out to anson@ada.support and shihan@ada.support if you have any questions whatsoever! This should be fun and not stressful.
- We intentionally left things somewhat ambiguous so that you can be creative. If you'd rather have things specified closely, we can give you more guidance. Just ask!
- Feel free to use any technology and programming language that you'd like

## Specifications

Your solution should start an HTTP service on any port you'd like. Please include instructions on how to start your service (so that we can test the functionality!)

### `/messages/` Resource

The `/messages/` resource accepts HTTP POST actions to create new messages in the conversation. A typical message resource has the format of:

```javascript
{
    "sender": "anson",
    "conversation_id": "1234",
    "message": "I'm a teapot"
}
```

Here, `"sender"` is a string username, `"conversation_id"` is a unique identifier for a particular conversation, and `"message"` is a string message to be logged to a conversation.

### `/conversations/<conversation_id>` Resource

The `/conversations/<conversation_id>` resource accepts an HTTP GET action and returns a list of conversation messages. A typical conversation as the format of:

```javascript
{
    "id": "1234",
    "messages": [
        {
            "sender": "anson",
            "message": "I'm a teapot",
            "created": "2018-01-17T04:50:14.883Z"
        },
        {
            "sender": "david",
            "message": "Short and stout",
            "created": "2018-01-17T04:52:17.201Z"
        }
    ]
}
```

Here, a conversation with two messages is presented.

## Clarifications
- Conversation IDs can follow any format you choose, as long as they are unique!
- Conversations should be persisted, but how you persist them is up to you :smile:
- You can assume that the entities sending incoming chat messages are authenticated and trustworthy (authentication is outside of the scope of this project)
- Don't worry about pagination on the conversations
- Don't worry about a list resource for conversations or messages
- We recommend validating incoming data
- Tests are always a good idea
- Please give us instructions on how to run your service when you open your Pull Request
