# Esto nos ayuda a posicionar nuestros config files en una carpeta dentro de nuestro proyecto
CONFIG_PATH=${HOME}/Downloads/proyecto_distribuido/test

.PHONY: init

init:
	if not exist "${CONFIG_PATH}" mkdir "${CONFIG_PATH}"

.PHONY: gencert
# gencert
# First creates the bare certificate, it is just the base certificate that others will differ from
# Then creates the server certificate, this allows our server certification
# Finally we create the client certificate this allows two way authentication
gencert:
	cfssl gencert \
		-initca test/ca-csr.json | cfssljson -bare ca

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=server \
		test/server-csr.json | cfssljson -bare server

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		test/client-csr.json | cfssljson -bare client

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="root" \
		test/client-csr.json | cfssljson -bare root-client

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="nobody" \
		test/client-csr.json | cfssljson -bare nobody-client
	
	move "C:\Users\helen\Downloads\proyecto_distribuido\*.csr" "C:\Users\helen\Downloads\proyecto_distribuido\test\"
	move "C:\Users\helen\Downloads\proyecto_distribuido\*.pem" "C:\Users\helen\Downloads\proyecto_distribuido\test\"

	copy "test\model.conf" "C:\Users\helen\Downloads\proyecto_distribuido\model.conf"

	copy "test\policy.csv" "C:\Users\helen\Downloads\proyecto_distribuido\policy.csv"


test:
	go test -race ./...

compile_rpc:
	protoc api/v1/*.proto \
	--go_out=. \
	--go_opt=paths=source_relative \
    --go-grpc_out=. \
	--go-grpc_opt=paths=source_relative \
	--proto_path=.