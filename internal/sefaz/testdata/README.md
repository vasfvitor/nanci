`evento-ciencia-signed.xml` is the golden signed evento, generated from the mock certificate with `go test ./internal/sefaz -run TestSignEvento_Golden -update`.

`retdist-cte-137.xml`, `retdist-cte-138.xml` and `retdist-cte-656.xml` are fictitious CTeDistribuicaoDFe answers. The two docZips of the 138 one are a minimal procCTe and a procEventoCTe (cancelamento) of the made-up key 35260970860312000150570010000001231123456788, gzipped and base64 encoded.
