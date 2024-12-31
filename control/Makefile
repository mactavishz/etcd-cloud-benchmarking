SRC = main.go
BIN = benchctl
INIT_ARGS = config init
RESET_ARGS = config reset
LOAD_ARGS = load
RUN_ARGS = run localhost:50051

.PHONY: all run load init reset clean

all: $(SRC)
	go build -o ../$(BIN) $(SRC)

config-init: all
	../$(BIN) $(INIT_ARGS)

config-reset: all
	../$(BIN) $(RESET_ARGS)

load: all
	../$(BIN) $(LOAD_ARGS)

run: all
	../$(BIN) $(RUN_ARGS)

clean:
	rm -f ../$(BIN)
