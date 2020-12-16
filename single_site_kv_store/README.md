# RESTful-key_value_store

The application uses gorilla/mux and has also been dockerized<br>
Use curl to access the key value store and perform any of the CRUD functions<br>

POST request : ```curl -d "value=<value>" -X POST http://localhost:8080/<key>```<br>
GET request : ```curl -X GET  http://localhost:8080/<key>```<br>
PUT request : ```curl -d "value=<value>" -X PUT http://localhost:8080/<key>```<br>
DELETE request : ```curl -X DELETE  http://localhost:8080/<key>```<br>

To build docker image use : ```docker build -t <dockerimage> .```<br>
To run docker container use : ```docker run -d -p 8080:8080 <dockerimage>```<br>