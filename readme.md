# Apple Push Notification Service Provider

[![Build Status](https://travis-ci.org/mdigger/pusher.svg?branch=master)](https://travis-ci.org/mdigger/pusher)

## POST /apns/com.xyzrd.trackintouch/users/dmitrys

+ Request (application/json; charset=utf-8)

    + Headers

            Authorization: Basic ZG1pdHJ5czo...bHMgKioqKio=

    + Body

            {
                "token": "EF2A1B9AF717...E6"
            }

+ Response 201 (application/json; charset=utf-8)

    + Headers

            X-Service-Version: 2.1.29
            X-Api-Version: 1.1

    + Body

            {
                "code": 201,
                "status": "Created",
                "success": true,
                "data": {
                    "tokens": [
                        "507C1666D7ECA6...A8FCCAAD5CEE580EE8C",
                        "6B0420FA3B631D...5F76BFA32862F284572",
                        "BE311B5BADA725...2D3D605840860FEBB28",
                        "EF2A1B9AF717B5...37442BF15CA9DE328E6"
                    ]
                }
            }
            


## GET /apns/com.xyzrd.trackintouch/users/dmitrys

+ Request (application/x-www-form-urlencoded; charset=utf-8)

    + Headers

            Authorization: Basic ZG1pdHJ5czo...bHMgKioqKio=



+ Response 200 (application/json; charset=utf-8)

    + Headers

            X-Service-Version: 2.1.29
            X-Api-Version: 1.1

    + Body

            {
                "code": 200,
                "status": "OK",
                "success": true,
                "data": {
                    "tokens": [
                        "507C1666D7ECA6...A8FCCAAD5CEE580EE8C",
                        "6B0420FA3B631D...5F76BFA32862F284572",
                        "BE311B5BADA725...2D3D605840860FEBB28",
                        "EF2A1B9AF717B5...37442BF15CA9DE328E6"
                    ]
                }
            }
            


## POST /apns/com.xyzrd.trackintouch/users/dmitrys/push

+ Request (application/json; charset=utf-8)

    + Headers

            Authorization: Basic ZG1pdHJ5czo...bHMgKioqKio=

    + Body

            {
                "payload": {
                    "aps": {
                        "alert": "Test message"
                    }
                },
                "expiration": "2016-10-30T00:00:00Z",
                "lowPriority": true,
                "collapseId": "cid"
            }

+ Response 200 (application/json; charset=utf-8)

    + Headers

            X-Service-Version: 2.1.29
            X-Api-Version: 1.1

    + Body

            {
                "code": 200,
                "status": "OK",
                "success": true,
                "data": {
                    "sent": {
                        "507C1666D7ECA6...A8FCCAAD5CEE580EE8C": "OK",
                        "6B0420FA3B631D...5F76BFA32862F284572": "OK",
                        "BE311B5BADA725...2D3D605840860FEBB28": "OK",
                        "EF2A1B9AF717B5...37442BF15CA9DE328E6": "OK"
                    }
                }
            }
            


## POST /apns/com.xyzrd.trackintouch/push

+ Request (application/json; charset=utf-8)

    + Headers

            Authorization: Basic ZG1pdHJ5czo...bHMgKioqKio=

    + Body

            {
                "payload": {
                    "aps": {
                        "alert": "Test message"
                    }
                },
                "users": [
                    "43892780306469875",
                    "dmitrys"
                ]
            }

+ Response 200 (application/json; charset=utf-8)

    + Headers

            X-Service-Version: 2.1.29
            X-Api-Version: 1.1

    + Body

            {
                "code": 200,
                "status": "OK",
                "success": true,
                "data": {
                    "sent": {
                        "507C1666D7ECA6...A8FCCAAD5CEE580EE8C": "OK",
                        "6B0420FA3B631D...5F76BFA32862F284572": "OK",
                        "BE311B5BADA725...2D3D605840860FEBB28": "OK"
                    }
                }
            }
            


