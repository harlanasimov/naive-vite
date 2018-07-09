# naive-vite
simple implement of vite


# Usage

one terminal, run main process:
```
> cd naive-vite
> go run main.go
```


and open another terminal, connect it using tcp:

> nc localhost 9000

1. choose node address
2. choose a role,  1:normal node, can send tx and receive tx.  2:snapshot node, just can generate snapshot block.
