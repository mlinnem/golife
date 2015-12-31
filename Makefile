#TODO: Need source files that go into tmp to be what's available via 'go get' command to follow Go best practices
FILES = cell.go cellpool.go constants.go logging.go utility.go worldstate.go

build:
	[ -d ./tmp ] || mkdir ./tmp
	cp *.go ./tmp ;
	cp ./lib/*.go ./tmp ;
	 #[ -f ./tmp/* ] || rm ./tmp/*
	 for file in $(FILES); do \
 		echo $$file ; \
		m4 -P -D "Debug=$$@" < ./tmp/$$file > ./tmp/$$file.tmpgo ; \
		mv ./tmp/$$file.tmpgo ./tmp/$$file ; \
 	done

	cp ./tmp/* ./lib2
	rm ./lib2/golife.go
	go build ./tmp/golife.go > ./golife

build_debug:
	[ -d tmp ] || mkdir ./tmp
	#[ -f ./tmp/* ] || rm ./tmp/*
	cp *.go ./tmp
	cp lib/*.go ./tmp
	for file in $(FILES); do \
		echo $$file ; \
		m4  -P -D "Debug=" < ./tmp/$$file > ./tmp/$$file.tmpgo ; \
		mv ./tmp/$$file.tmpgo ./tmp/$$file ; \
	done
	cp ./tmp/* ./lib2
	rm ./lib2/golife.go
	go build ./tmp/golife.go  > ./golife

run: build
		./golife

run_debug: build_debug
		./golife

clean:
	rm golife
	rm tmp/*
	rmdir tmp
	mv *.pdf performance_testruns/
