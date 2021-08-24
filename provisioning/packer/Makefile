default: generate

files-generated:  files-generated/REVISION files-generated/isucon11-final.tar files-generated/benchmarker

files-generated/REVISION:
	mkdir -p files-generated/
	if [ -n "$(GITHUB_SHA)" ]; then echo "$(GITHUB_SHA)" > files-generated/REVISION; fi
	if [ ! -e files-generated/REVISION ]; then git rev-parse HEAD > files-generated/REVISION; fi
	test -e files-generated/REVISION

files-generated/isucon11-final.tar: files-generated/REVISION
	mkdir -p files-generated/
	git -C "$(shell git rev-parse --show-toplevel)" archive "$(shell cat files-generated/REVISION)" > files-generated/isucon11-final.tar

files-generated/benchmarker:
	mkdir -p files-generated/
	cd ../../benchmarker && make
	mv ../../benchmarker/bin/benchmarker files-generated/benchmarker

generate: output target/contestant.json target/bench.json target/standalone.json

output:
	mkdir -p output/

target/contestant.json: files-generated output base.libsonnet contestant.jsonnet
	mkdir -p target/
	jsonnet --jpath . contestant.jsonnet > target/contestant.json
	packer validate target/contestant.json

target/bench.json: files-generated output base.libsonnet bench.jsonnet
	mkdir -p target/
	jsonnet --jpath . bench.jsonnet > target/bench.json
	packer validate target/bench.json

target/standalone.json: files-generated base.libsonnet standalone.jsonnet
	mkdir -p target/
	jsonnet --jpath . standalone.jsonnet > target/standalone.json
	packer validate target/standalone.json


build: build-contestant build-bench

build-contestant: files-generated target/contestant.json
	packer build -var "revision=$(shell cat files-generated/REVISION)" target/contestant.json
build-bench: files-generated target/bench.json
	packer build -var "revision=$(shell cat files-generated/REVISION)" target/bench.json
build-standalone: files-generated target/standalone.json
	packer build -var "revision=$(shell cat files-generated/REVISION)" target/standalone.json


clean:
	rm -rf files-generated/
	rm -rf target/

clean-output:
	rm -rf output/
