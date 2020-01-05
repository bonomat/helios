# Helios

1. Start first peer

    ```go
    go run github.com/bonomat/helios -listen /ip4/127.0.0.1/tcp/5551
    ```

    You will find a connection string on the stdout after starting helios. e.g.

    ```bash
    /ip4/127.0.0.1/tcp/5551/p2p/16Uiu2HAmKTFF3shwGL1g4U6dCzdoLvP7F1T9b3Y1w13Ybz1eQdYw
    ```

1. Connect second peer

    ```go
    go run github.com/bonomat/helios -listen /ip4/127.0.0.1/tcp/5552 -peer /ip4/127.0.0.1/tcp/5551/p2p/16Uiu2HAmKTFF3shwGL1g4U6dCzdoLvP7F1T9b3Y1w13Ybz1eQdYw
    ```

1. Connect as many peers as you like