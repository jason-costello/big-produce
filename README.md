---
Requirements and assumptions
---

# Project Overview

Version 1 Requirement‌

* The API is RESTful
* Error handling \(GET nonexistent produce, bad POST payload, etc.\)
* The produce unit price is a number with up to 2 decimal places
* The produce codes are alphanumeric and case-insensitive
* The produce codes are sixteen characters long, with dashes separating each four character group
* The produce name is alphanumeric and case-insensitive
* The produce includes name, produce code, and unit price
* Supports adding more than one new produce item at a time
* The produce database is a single, in memory array of data and supports reads and writes
* The API supports adding and deleting individual produce. You can also get any produce in the database.
* The functionality is tested and verifiable without manually exercising the endpoints Get all the produce in the database. Return JSON array of produce. Ensure that you can get an individual item from the API too.

  ​

## Assumptions made during design‌ <a id="assumptions-made-during-design"></a>

The documentation states that only alphanumeric, but two of the four produce items given to load the db with have spaces in their names. Assuming the company does not want the names modified to not include a space, I have added a space as a valid character.‌

Auth is not mentioned in the document and I didn't want to include anything as I am not sure if it would impact any type of testing your team has configured for this project. If I were to implement auth for this my first attempt would be to utilize a token in a header such as X-Auth-Header key. Best case scenario, this key is generated through a service I am not responsible for. Worst case scenario, I'm writing an authentication service that would be external to this application.‌

There is a discrepancy between the user stories and the acceptance criteria. The user stories state only add, fetch all, fetch one are required while the acceptance criteria state that deleting individual items is required. Not having this functionality would be odd and seems to be something the user would request in the very near future so I have included a delete function with the application.‌

The ability to add one or more item is present through the same endpoint. To add one item to the db, it would need to be inside an array in the post body. The decision to do this was to eliminate an endpoint dedicated to just adding a single item.



# API Overview

## Base url

The base url used to access all endpoints:  
`http://www.bigproduce.com/api`

## Authentication

The Big Produce team believes in an open community commited to documenting the finest produce specimines for sale across the globe.   Because we allow anyone to become a partnered seller we feel authentication to be an overreach.  We are also opposed to security via obscurity so accessing our endpoints are as easy as visiting any website!

## Versioning

The APIs are versioned and it is possible for each endpoint to have a version unique from the others.  
To specify a version use `v1`, `v2`, `v3`, etc on the base url.

Examples:   
`http://www.bigproduce.com/api/v1/produce`  
`http://www.bigproduce.com/api/v2/produce`

## Rate limiting

Our rate limiting philosophy is inline with our authentication policy; we do not implement rate limiting.

## Request Timeout

There is a five second request timout across all endpoints.   This is a hard-coded value that could be modified in the future if there is demand for a change from the community.   If there are many changes, I suppose we could get the developers to implement this value as an env variable to eliminate the need for code changes in the future.

## Status codes

Standard HTTP status codes are returned.  If all goes well you should only see the first three, but the potential is there for you to experience all of the below.

200 - success  
201 - added  
204 - deleted  
400 - bad request  
409 - item already exists  
500 - internal server error \(problem is on our side, not yours\)

## Default DB records

The following records created at startup are the default records in the database.

```javascript
[ { "produce_name": "Lettuce", "produce_code": "A12T-4GH7-QPL9-3N4M", "produce_unit_price": 3.46 }, { "produce_name": "Peach", "produce_code": "E5T6-9UI3-TH15-QR88", "produce_unit_price": 2.99 }, { "produce_name": "Green Pepper", "produce_code": "YRT6-72AS-K736-L4AR", "produce_unit_price": 0.79 }, { "produce_name": "Gala Apple", "produce_code": "TQ4C-VV6T-75ZX-1RMR", "produce_unit_price": 3.59 } ]
```



---
Build scratch container for the app
---

# Dockerfile

## Dockerfile used by github actions to build the container

```bash
FROM golang:1.17.0-alpine3.13 AS builder
RUN apk update && apk add ca-certificates

ADD ./ /appdir/
RUN cd /appdir && \
    go mod tidy && \
    go mod vendor && \
    GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -a -ldflags=-X=main.version=${VERSION} -tags netgo -ldflags="-w -s" -o app

## Build scratch container and only copy over binary and certs
FROM scratch
COPY --from=builder /appdir/app /usr/local/bin/app

USER 1001
EXPOSE :8088
ENTRYPOINT [ "app" ]
```



---
GitHub Actions are used to test and build a containerized app.
---

# Build & Deploy

## github actions

In order to initiate the action, a tag starting with 'v' has to be pushed to any branch.

The build step runs `go build -v ./...` and `go test -v ./...`   .

The deploy step builds a scratch container with the application and pushes it to the github container registery.  To access the container the url below can be used:  
`ghcr.io/jason-costello/silver-octo-carnival:latest`

We are working towards a properly versioned label for the container but are working through an issue that cause a failure when attempting to apply the following tag to the container:

```bash
ghcr.io/${{ github.repository }}:${{ github.ref }}
```
