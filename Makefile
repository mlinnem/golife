FILES:\
	cell.go \
	cellpool.go \
	constants.go \
	logging.go \
	utility.go \
	worldstate.go

build:
	 cp *.go ./tmp
	 cp lib/*.go ./tmp
	 @- $(foreach FILE,$(FILES), \
	 m4 -D 'Debug=$@' < ./tmp/$FILE > ./tmp/$FILE
	 )
	 go build tmp/golife.go > ./golife

build_debug:
	cp *.go ./tmp
	cp lib/*.go ./tmp
	@- $(foreach FILE,$(FILES), \
	m4 -D 'Debug=$@' < ./tmp/$FILE > ./tmp/$FILE
	)
	go build tmp/golife.go > ./golife

run: build
		./golife

run_debug: build_debug
		./golife

clean:
	rm golife
	rm golife_expanded.go
