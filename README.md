# naive-vite
simple implement of vite


# Usage

one terminal, run main process:
```
> cd naive-vite/cmd/example_v0_1
> go run main.go
```


and open another terminal, connect it using tcp:

> nc localhost 9000

1. input node address
2. input a role,  1:normal node, can send tx and receive tx.  2:snapshot node, just can generate snapshot block.
