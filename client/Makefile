SRC = main.go
BIN = benchclient

.PHONY: all run clean

all: $(SRC)
	go build -o ../$(BIN) $(SRC)

run: all
	../$(BIN)

clean:
	rm -f ../$(BIN)
	rm -f *metrics.json
	rm -f *.log
