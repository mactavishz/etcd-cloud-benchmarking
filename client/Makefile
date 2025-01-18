SRC = main.go
BUILD = ../bin
BIN = benchclient

.PHONY: all run clean

all: $(SRC)
	go build -o $(BUILD)/$(BIN) $(SRC)

run: all
	../$(BIN)

clean:
	rm -f $(BUILD)/$(BIN)
	rm -f $(GOPATH)/bin/$(BIN)
	rm -f *metrics.csv
	rm -f keys.txt
	rm -f *.log
