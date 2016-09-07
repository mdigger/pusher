# Apple Push Notification Service Provider

## Service params

	$ ./pusher2 --help
	# Pusher 2.0.25 [git 38cabcb] (2016-09-01)
	Usage of ./pusher2:
	  -addr port
	       	http server address and port (default ":8443")
	  -cert file
	       	server certificate file (default "cert.pem")
	  -compress
	       	gzip compress response (default true)
	  -config file
	       	configuration file (default "config.gob")
	  -indent
	       	indent JSON response (default true)
	  -key file
	       	server private certificate file (default "key.pem")
	  -monitor
	       	start monitor handler
	  -pools size
	       	APNS client pool size (default 1)
	  -reset
	       	remover users and admin authorization
	  -store DSN
	       	db DSN connection string (default "tokens.db")


## Administration

Store admin authorization in variable `basicAdminAuth`:

	basicAdminAuth="admin:password"


### Admin

- Get admin status

		$ curl -k -i "https://localhost:8443/admin" -u $basicAdminAuth

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 24

		{
		    "secured": true
		}

- Remove admin

		$ curl -k -i -X "DELETE" "https://localhost:8443/admin" -u $basicAdminAuth 

		HTTP/2 204

-  Set admin login & password

		$ curl -k -i -X "POST" "https://localhost:8443/admin" -u $basicAdminAuth \
			--data-urlencode "login=admin" \
			--data-urlencode "password=admin_password"

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 24

		{
		    "secured": true
		}


### Users

-  Add user

		$ curl -k -i -X "POST" "https://localhost:8443/users" -u $basicAdminAuth \
			--data-urlencode "login=login" \
			--data-urlencode "password=password"

		HTTP/2 204
		location: /users/login

-  Add user (JSON)

		$ curl -k -i -X "POST" "https://localhost:8443/users" -u $basicAdminAuth \
			-H "Content-Type: application/json; charset=utf-8" \
			-d "{\"login\":\"login2\",\"password\":\"password2\"}"

		HTTP/2 201
		location: /users/login2
		content-type: text/plain; charset=utf-8
		content-length: 0

-  Get users list

		$ curl -k -i "https://localhost:8443/users" -u $basicAdminAuth \

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 59

		{
		    "users": [
		        "login",
		        "login2"
		    ]
		}

-  Change user password

		$ curl -k -i -X "PUT" "https://localhost:8443/users/login2" \
			-u $basicAdminAuth \
			--data-urlencode "password=new_password"

		HTTP/2 204

-  Delete user

		$ curl -k -i -X "DELETE" "https://localhost:8443/users/login2" \
			-u $basicAdminAuth

		HTTP/2 204


### Certificates

-  Upload certificate

		$ curl -k -i -X "POST" "https://localhost:8443/certificates" \
			-u $basicAdminAuth \
			-F "certificate=@cert.p12" \
			-F "password=xopen123"

		HTTP/2 200
		content-type: application/json; charset=utf-8
		location: /certificates/com.xyzrd.trackintouch
		content-length: 438

		{
		    "CName": "Apple Push Services: com.xyzrd.trackintouch",
		    "OrgName": "XYZRD GROUP OU",
		    "OrgUnit": "W23G28NPJW",
		    "Country": "US",
		    "BundleID": "com.xyzrd.trackintouch",
		    "Topics": [
		        "com.xyzrd.trackintouch",
		        "com.xyzrd.trackintouch.voip",
		        "com.xyzrd.trackintouch.complication"
		    ],
		    "Development": true,
		    "Production": true,
		    "IsApple": true,
		    "Expire": "2017-06-26T06:05:40Z"
		}

    	#### Expired

		$ curl -k -i -X "POST" "https://localhost:8443/certificates" \
			-u $basicAdminAuth \
			-F "certificate=@expired.p12" \
			-F "password=xopen123"

		HTTP/2 400
		content-type: application/json; charset=utf-8
		content-length: 48

		{
		  "code": 400,
		  "error": "x509: certificate has expired or is not yet valid"
		}

    	#### Bad

		$ curl -k -i -X "POST" "https://localhost:8443/certificates" \
			-u $basicAdminAuth \
			-F "certificate=@bad.p12" \
			-F "password=xopen123"

		HTTP/2 400
		content-type: application/json; charset=utf-8
		content-length: 86

		{
		  "code": 400,
		  "error": "pkcs12: expected exactly two safe bags in the PFX PDU"
		}

-  Upload certificate (base64)

		$ curl -k -i -X "POST" "https://localhost:8443/certificates" -u $basicAdminAuth \
			--data-urlencode "certificate=$(base64 cert2.p12)" \
			--data-urlencode "password=xopen123"

		HTTP/2 200
		content-type: application/json; charset=utf-8
		location: /certificates/com.xyzrd.messagetrack.ios
		content-length: 458

		{
		    "CName": "Apple Push Services: com.xyzrd.messagetrack.ios",
		    "OrgName": "XYZRD GROUP OU",
		    "OrgUnit": "W23G28NPJW",
		    "Country": "US",
		    "BundleID": "com.xyzrd.messagetrack.ios",
		    "Topics": [
		        "com.xyzrd.messagetrack.ios",
		        "com.xyzrd.messagetrack.ios.voip",
		        "com.xyzrd.messagetrack.ios.complication"
		    ],
		    "Development": true,
		    "Production": true,
		    "IsApple": true,
		    "Expire": "2017-02-09T19:51:58Z"
		}

-  Upload certificate (JSON)

		$ curl -k -i -X "POST" "https://localhost:8443/certificates" -u $basicAdminAuth \
			-H "Content-Type: application/json; charset=utf-8" \
			-d "{\"certificate\":\"$(base64 cert4.p12)\",\"password\":\"xopen123\"}"

		HTTP/2 201
		content-type: application/json; charset=utf-8
		location: /certificates/com.xyzrd.trackintouch.kid
		content-length: 458

		{
		    "CName": "Apple Push Services: com.xyzrd.trackintouch.kid",
		    "OrgName": "XYZRD GROUP OU",
		    "OrgUnit": "W23G28NPJW",
		    "Country": "US",
		    "BundleID": "com.xyzrd.trackintouch.kid",
		    "Topics": [
		        "com.xyzrd.trackintouch.kid",
		        "com.xyzrd.trackintouch.kid.voip",
		        "com.xyzrd.trackintouch.kid.complication"
		    ],
		    "Development": true,
		    "Production": true,
		    "IsApple": true,
		    "Expire": "2017-02-10T13:01:22Z"
		}

-  Get certificate info

		$ curl -k -i "https://localhost:8443/certificates/com.xyzrd.trackintouch" -u $basicAdminAuth

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 438

		{
		    "CName": "Apple Push Services: com.xyzrd.trackintouch",
		    "OrgName": "XYZRD GROUP OU",
		    "OrgUnit": "W23G28NPJW",
		    "Country": "US",
		    "BundleID": "com.xyzrd.trackintouch",
		    "Topics": [
		        "com.xyzrd.trackintouch",
		        "com.xyzrd.trackintouch.voip",
		        "com.xyzrd.trackintouch.complication"
		    ],
		    "Development": true,
		    "Production": true,
		    "IsApple": true,
		    "Expire": "2017-06-26T06:05:40Z"
		}

-  Get certificates list

		$ curl -k -i "https://localhost:8443/certificates" -u $basicAdminAuth

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 141

		{
		    "certificates": [
		        "com.xyzrd.messagetrack.ios",
		        "com.xyzrd.trackintouch",
		        "com.xyzrd.trackintouch.kid"
		    ]
		}

-  Delete certificate

		$ curl -k -i -X "DELETE" "https://localhost:8443/certificates/com.xyzrd.trackintouch.kid" -u $basicAdminAuth

		HTTP/2 204


## Topics & Push

Create basic authorization header in variable `basicAuth`:

	basicAuth="login:password"

### Topics

- Get topics list


		$ curl -k -i "https://localhost:8443/apns" -u $basicAuth

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 277

		{
		    "topics": [
		        "com.xyzrd.messagetrack.ios",
		        "com.xyzrd.messagetrack.ios.complication",
		        "com.xyzrd.messagetrack.ios.voip",
		        "com.xyzrd.trackintouch",
		        "com.xyzrd.trackintouch.complication",
		        "com.xyzrd.trackintouch.voip"
		    ]
		}

- Get topic info

		$ curl -k -i "https://localhost:8443/apns/com.xyzrd.trackintouch" -u $basicAuth

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 438

		{
		    "CName": "Apple Push Services: com.xyzrd.trackintouch",
		    "OrgName": "XYZRD GROUP OU",
		    "OrgUnit": "W23G28NPJW",
		    "Country": "US",
		    "BundleID": "com.xyzrd.trackintouch",
		    "Topics": [
		        "com.xyzrd.trackintouch",
		        "com.xyzrd.trackintouch.voip",
		        "com.xyzrd.trackintouch.complication"
		    ],
		    "Development": true,
		    "Production": true,
		    "IsApple": true,
		    "Expire": "2017-06-26T06:05:40Z"
		}


### Tokens

- Add user token

		$ curl -k -i -X "POST" "https://localhost:8443/apns/com.xyzrd.trackintouch/users/dmitrys" -u $basicAuth \
			--data-urlencode "token=BE311B5BADA725B323B1A56E03ED25B4814D6B9EDF5B02D3D605840860FEBB28"

		HTTP/2 204

- Add user token [JSON]

		$ curl -k -i -X "POST" "https://localhost:8443/apns/com.xyzrd.trackintouch/users/dmitrys" -u $basicAuth \
			-H "Content-Type: application/json; charset=utf-8" \
			-d "{\"token\":\"507C1666D7ECA6C26F40BC322A35CCB937E2BF02DFDACA8FCCAAD5CEE580EE8C\"}"

		HTTP/2 204

- Add user token [multipart]

		$ curl -k -i -X "POST" "https://localhost:8443/apns/com.xyzrd.trackintouch/users/dmitrys" -u $basicAuth \
			-F "token=6B0420FA3B631DF5C13FB9DDC1BE8131C52B4E02580BB5F76BFA32862F284572"

		HTTP/2 204

- Get users list

		$ curl -k -i "https://localhost:8443/apns/com.xyzrd.trackintouch/users" -u $basicAuth

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 43

		{
		    "users": [
		        "dmitrys"
		    ]
		}

- Get user tokens list

		$ curl -k -i "https://localhost:8443/apns/com.xyzrd.trackintouch/users/dmitrys" -u $basicAuth

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 253

		{
		    "tokens": [
		        "507C1666D7ECA6C26F40BC322A35CCB937E2BF02DFDACA8FCCAAD5CEE580EE8C",
		        "6B0420FA3B631DF5C13FB9DDC1BE8131C52B4E02580BB5F76BFA32862F284572",
		        "BE311B5BADA725B323B1A56E03ED25B4814D6B9EDF5B02D3D605840860FEBB28"
		    ]
		}


### Push

- Push to all user devices

		$ curl -k -i -X "POST" "https://localhost:8443/apns/com.xyzrd.trackintouch/users/dmitrys/push" -u $basicAuth \
			--data-urlencode "payload={\"aps\":{\"alert\":\"APNS user test message\"}}" \
			--data-urlencode "expiration=2016-08-31T16:34:28Z" \
			--data-urlencode "lowPriority=on"

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 18

		{
		    "push": 3
		}

- Push to all user devices [JSON]

		$ curl -k -i -X "POST" "https://localhost:8443/apns/com.xyzrd.trackintouch/users/dmitrys/push" -u $basicAuth \
			-H "Content-Type: application/json; charset=utf-8" \
			-d "{\"payload\":\"{\\\"aps\\\":{\\\"alert\\\":\\\"APNS user test message 2\\\"}}\",\"expiration\":\"2016-08-31T16:34:28Z\",\"lowPriority\":true}"

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 18

		{
		    "push": 3
		}

- Push to users

		$ curl -k -i -X "POST" "https://localhost:8443/apns/com.xyzrd.trackintouch/push" -u $basicAuth \
			--data-urlencode "payload={\"aps\":{\"alert\":\"APNS user test message\"}}" \
			--data-urlencode "lowPriority=true" \
			--data-urlencode "expiration=2016-08-31T16:34:28Z" \
			--data-urlencode "user=dmitrys" \
			--data-urlencode "user=test"

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 64

		{
		    "push": {
		        "dmitrys": 3,
		        "test": 0
		    }
		}

- Push to users [JSON]

		$ curl -k -i -X "POST" "https://localhost:8443/apns/com.xyzrd.trackintouch/push" -u $basicAuth \
			-H "Content-Type: application/json; charset=utf-8" \
			-d "{\"payload\":{\"aps\":{\"alert\":\"APNS all users push test\"}},\"expiration\":\"2016-08-31T16:34:28Z\",\"lowPriority\":true,\"users\":[\"dmitrys\",\"test\"]}"

		HTTP/2 200
		content-type: application/json; charset=utf-8
		content-length: 64

		{
		    "sent": {
		        "dmitrys": 3,
		        "test": 0
		    }
		}

## Log

Access log (`stderr`):

	2016/08/31 20:08:48 127.0.0.1          135.509µs     0.024Kb 200     GET /admin
	2016/08/31 20:08:48 127.0.0.1          432.314µs     0.000Kb 204  DELETE /admin
	2016/08/31 20:08:49 127.0.0.1           628.53µs     0.023Kb 200    POST /admin
	2016/08/31 20:08:50 127.0.0.1          546.106µs     0.000Kb 201    POST /users
	2016/08/31 20:08:50 127.0.0.1          571.342µs     0.000Kb 201    POST /users
	2016/08/31 20:08:51 127.0.0.1           156.24µs     0.058Kb 200     GET /users
	2016/08/31 20:08:52 127.0.0.1          891.372µs     0.000Kb 204     PUT /users/login2
	2016/08/31 20:08:53 127.0.0.1          475.399µs     0.000Kb 204  DELETE /users/login2
	2016/08/31 20:08:54 127.0.0.1       188.558597ms     0.428Kb 201    POST /certificates
	2016/08/31 20:08:55 127.0.0.1        20.837984ms     0.447Kb 201    POST /certificates
	2016/08/31 20:08:56 127.0.0.1        21.869238ms     0.447Kb 201    POST /certificates
	2016/08/31 20:08:56 127.0.0.1          227.495µs     0.428Kb 200     GET /certificates/com.xyzrd.trackintouch
	2016/08/31 20:08:56 127.0.0.1          143.311µs     0.138Kb 200     GET /certificates
	2016/08/31 20:08:57 127.0.0.1          800.622µs     0.000Kb 204  DELETE /certificates/com.xyzrd.trackintouch.kid
	2016/08/31 20:08:58 127.0.0.1           157.97µs     0.271Kb 200     GET /apns
	2016/08/31 20:08:58 127.0.0.1          216.794µs     0.428Kb 200     GET /apns/com.xyzrd.trackintouch
	2016/08/31 20:08:59 127.0.0.1         1.140562ms     0.000Kb 201    POST /apns/com.xyzrd.trackintouch/users/dmitrys
	2016/08/31 20:08:59 127.0.0.1         1.351478ms     0.000Kb 201    POST /apns/com.xyzrd.trackintouch/users/dmitrys
	2016/08/31 20:09:00 127.0.0.1         2.742265ms     0.000Kb 201    POST /apns/com.xyzrd.trackintouch/users/dmitrys
	2016/08/31 20:09:01 127.0.0.1          821.581µs     0.042Kb 200     GET /apns/com.xyzrd.trackintouch/users
	2016/08/31 20:09:01 127.0.0.1          748.779µs     0.247Kb 200     GET /apns/com.xyzrd.trackintouch/users/dmitrys
	2016/08/31 20:09:03 127.0.0.1       1.600974989s     0.018Kb 200    POST /apns/com.xyzrd.trackintouch/users/dmitrys/push
	2016/08/31 20:09:04 127.0.0.1       519.384996ms     0.018Kb 200    POST /apns/com.xyzrd.trackintouch/users/dmitrys/push
	2016/08/31 20:09:05 127.0.0.1        523.27135ms     0.062Kb 200    POST /apns/com.xyzrd.trackintouch/push
	2016/08/31 20:09:06 127.0.0.1        516.28095ms     0.062Kb 200    POST /apns/com.xyzrd.trackintouch/push

Push log (`stdout`):

	2016/09/01 17:21:19 200 com.xyzrd.trackintouch dmitrys 507C1666D7ECA6C26F40BC322A35CCB937E2BF02DFDACA8FCCAAD5CEE580EE8C 36B50F76-28E2-F2AD-CFEE-DDAECE63353E
	2016/09/01 17:21:19 200 com.xyzrd.trackintouch dmitrys 6B0420FA3B631DF5C13FB9DDC1BE8131C52B4E02580BB5F76BFA32862F284572 802EA1CF-C91C-33C1-1AD6-379998F23CD1
	2016/09/01 17:21:19 200 com.xyzrd.trackintouch dmitrys BE311B5BADA725B323B1A56E03ED25B4814D6B9EDF5B02D3D605840860FEBB28 F989CE5E-EAEA-566F-A416-F76E1D439E1D
	2016/09/01 17:21:19 400 com.xyzrd.trackintouch dmitrys AF14781EFB37CD148802945F24DF34A5925E810AADEF46AA811328AE19D3D3D3 242C2685-0340-2CF0-9D53-73ECC90AA8DF Unregistered
	2016/09/01 17:21:19 400 com.xyzrd.trackintouch dmitrys AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA FCFBB32B-A32C-94F8-CB74-9C578F63DB5F BadDeviceToken