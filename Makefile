
.PHONY: mongoup
mongo-up:
	docker-compose -f docker-compose.mongo.yml \
	               up --build --abort-on-container-exit

.PHONY: up
up:
	docker-compose -f docker-compose.yml \
                   -f docker-compose.mongo.yml \
	               up --build --abort-on-container-exit

.PHONY: down
down:
	docker-compose -f docker-compose.yml \
                   -f docker-compose.mongo.yml \
	               down