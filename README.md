# Hooky

Hooky is a RESTful microservice to manage asynchronous tasks in the cloud.

It uses WebHooks to trigger tasks that can be executed immediately or at a regular interval.

Hooky offers similar services as:
 - Google Cloud Task Queue (push mode)
 - Google Cloud Scheduled Tasks
 - Microsoft Azure Scheduler

## Install

Hooky uses MongoDB as its storage backend, by default it expects MongoDB to be running on localhost and will listen on HTTP port 8000.

```
$ go get github.com/sebest/hooky/cmd/hookyd
$ hookyd
```

## Features

- [x] RESTful API
- [x] Asynchronous tasks using [Webhooks](http://en.wikipedia.org/wiki/Webhook)
- [x] Tasks can be scheduled with recurrency using a CRON syntax
- [x] Configurable `retry` policy
- [x] Multi-accounts and multi-applications
- [X] Concurrency limit per Queue
- [X] Stats per Task
- [ ] Stats per Queue
- [ ] Stats per Application
- [ ] Crontabs
- [ ] Delayed Tasks
- [ ] Full documentation
- [ ] Tests

## RESTful API

The full API specification is [here]( https://raw.githubusercontent.com/sebest/hooky/master/swagger.yml) using the Swagger specification version 2.0.

You can visualize it [here](http://editor.swagger.io/#/edit?import=https://raw.githubusercontent.com/sebest/hooky/master/swagger.yml).

## Tutorial

For this tutorial we will use [httpie](https://github.com/jakubroztocil/httpie).

Hooky uses basic authentication, there is a default `admin` account with a default password that you use to create new accounts.

The default `admin` password is `admin`.

### Create a new account

```
$ http -v -a admin:admin POST :8000/accounts
POST /accounts HTTP/1.1
Accept: */*
Accept-Encoding: gzip, deflate
Authorization: Basic YWRtaW46YWRtaW4=
Connection: keep-alive
Content-Length: 0
Host: localhost:8000
User-Agent: HTTPie/0.9.2



HTTP/1.1 200 OK
Content-Length: 120
Content-Type: application/json
Date: Sun, 17 May 2015 00:24:30 GMT
X-Powered-By: go-json-rest

{
    "created": "2015-05-17T00:24:30Z",
    "id": "5557dd8eef015fb521000009",
    "key": "Ci6wgzetYviXHhJME6KvyNqkRCZjFBoe"
}
```

You now have an account `id` and `key` to connect to the service. When a new account is created a `default` application is automatically created for convenience.

### Create a new task

Using our new account we can now create a task, the only required parameter is `url`.

```
$ http -v -a 5557dd8eef015fb521000009:Ci6wgzetYviXHhJME6KvyNqkRCZjFBoe POST :8000/accounts/5557dd8eef015fb521000009/applications/default/tasks url=http://www.perdu.com
POST /accounts/5557dd8eef015fb521000009/applications/default/tasks HTTP/1.1
Accept: application/json
Accept-Encoding: gzip, deflate
Authorization: Basic NTU1N2RkOGVlZjAxNWZiNTIxMDAwMDA5OkNpNndnemV0WXZpWEhoSk1FNkt2eU5xa1JDWmpGQm9l
Connection: keep-alive
Content-Length: 31
Content-Type: application/json
Host: localhost:8000
User-Agent: HTTPie/0.9.2

{
    "url": "http://www.perdu.com"
}

HTTP/1.1 200 OK
Content-Length: 545
Content-Type: application/json
Date: Sun, 17 May 2015 00:27:39 GMT
X-Powered-By: go-json-rest

{
    "account": "5557dd8eef015fb521000009",
    "active": true,
    "application": "default",
    "at": "2015-05-17T00:27:39Z",
    "auth": {
        "password": "",
        "username": ""
    },
    "created": "2015-05-17T00:27:39Z",
    "errorRate": 0,
    "errors": 0,
    "executions": 0,
    "id": "5557e07bef015fb521000011",
    "method": "POST",
    "name": "5557e07bef015fb521000011",
    "queue": "default",
    "retry": {
        "attempts": 0,
        "factor": 2,
        "max": 300,
        "maxAttempts": 10,
        "min": 10
    },
    "status": "pending",
    "url": "http://www.perdu.com"
}
```

Our task is queued and will be executed immediately, as we did not provide a `name`, the task `id` is used as the task `name`.

We can query the status of our task:

```
$ http -v -a 5557dd8eef015fb521000009:Ci6wgzetYviXHhJME6KvyNqkRCZjFBoe GET :8000/accounts/5557dd8eef015fb521000009/applications/default/tasks/5557e07bef015fb521000011
GET /accounts/5557dd8eef015fb521000009/applications/default/tasks/5557e07bef015fb521000011 HTTP/1.1
Accept: */*
Accept-Encoding: gzip, deflate
Authorization: Basic NTU1N2RkOGVlZjAxNWZiNTIxMDAwMDA5OkNpNndnemV0WXZpWEhoSk1FNkt2eU5xa1JDWmpGQm9l
Connection: keep-alive
Host: localhost:8000
User-Agent: HTTPie/0.9.2



HTTP/1.1 200 OK
Content-Length: 593
Content-Type: application/json
Date: Sun, 17 May 2015 00:32:19 GMT
X-Powered-By: go-json-rest

{
    "account": "5557dd8eef015fb521000009",
    "active": false,
    "application": "default",
    "auth": {
        "password": "",
        "username": ""
    },
    "created": "2015-05-17T00:27:39Z",
    "errorRate": 0,
    "errors": 0,
    "executed": "2015-05-17T00:27:40Z",
    "executions": 1,
    "id": "5557e07bef015fb521000011",
    "lastSuccess": "2015-05-17T00:27:40Z",
    "method": "POST",
    "name": "5557e07bef015fb521000011",
    "queue": "default",
    "retry": {
        "attempts": 0,
        "factor": 2,
        "max": 300,
        "maxAttempts": 10,
        "min": 10
    },
    "status": "success",
    "url": "http://www.perdu.com"
}
```

Every tasks generate `attempts`, if the task succeed on its first attempt, you will only have one, otherwise you can have up to 10 attempts by default.

You can get the attempts' list:

```
$ http -v -a 5557dd8eef015fb521000009:Ci6wgzetYviXHhJME6KvyNqkRCZjFBoe GET :8000/accounts/5557dd8eef015fb521000009/applications/default/tasks/5557e07bef015fb521000011/attempts
GET /accounts/5557dd8eef015fb521000009/applications/default/tasks/5557e07bef015fb521000011/attempts HTTP/1.1
Accept: */*
Accept-Encoding: gzip, deflate
Authorization: Basic NTU1N2RkOGVlZjAxNWZiNTIxMDAwMDA5OkNpNndnemV0WXZpWEhoSk1FNkt2eU5xa1JDWmpGQm9l
Connection: keep-alive
Host: localhost:8000
User-Agent: HTTPie/0.9.2



HTTP/1.1 200 OK
Content-Length: 593
Content-Type: application/json
Date: Sun, 17 May 2015 00:34:34 GMT
X-Powered-By: go-json-rest

{
    "count": 1,
    "hasMore": false,
    "list": [
        {
            "account": "5557dd8eef015fb521000009",
            "application": "default",
            "auth": {
                "password": "",
                "username": ""
            },
            "created": "2015-05-17T00:27:39Z",
            "id": "5557e07bef015fb521000012",
            "method": "POST",
            "name": "5557e07bef015fb521000011",
            "queue": "default",
            "status": "success",
            "statusCode": 200,
            "statusMessage": "200 OK",
            "taskID": "5557e07bef015fb521000011",
            "url": "http://www.perdu.com"
        }
    ],
    "page": 1,
    "pages": 1,
    "total": 1
}
```
