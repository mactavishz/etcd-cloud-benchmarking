SRC = main.go
BIN = benchctl
RUN_ARGS = load 10000

.PHONY: all run clean

all: $(SRC)
	go build -o $(BIN) $(SRC)

run: all
	./$(BIN) $(RUN_ARGS)

clean:
	rm -f $(BIN)
