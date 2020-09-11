all:
	go build

run: all
	./bpc

install:
	go install

clean:
	rm -f bpc
	rm -f *.exe