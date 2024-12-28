SRC = main.go
BIN = benchctl
LOAD_ARGS = load 10000
RUN_ARGS = run

.PHONY: all run clean

all: $(SRC)
	go build -o ../$(BIN) $(SRC)

load: all
	../$(BIN) $(LOAD_ARGS)

run: all
	../$(BIN) $(RUN_ARGS)

clean:
	rm -f ../$(BIN)
