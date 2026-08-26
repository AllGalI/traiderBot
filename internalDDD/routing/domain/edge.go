
for f in \
	internalDDD/market/application/market_data_service.go \
	internalDDD/market/application/market_data_service_test.go \
	internalDDD/market/application/market_graph_service.go \
	internalDDD/market/application/market_graph_service_test.go \
	internalDDD/market/application/market_structure_service.go \
	internalDDD/market/application/market_structure_service_test.go \
	internalDDD/market/application/orderbook_stream.go \
	internalDDD/market/application/symbol_provider.go 
do
  echo "===== $f ====="
  cat "$f"
done

for f in \
	internalDDD/market/domain/fee_policy.go \
	internalDDD/market/domain/graph.go \
	internalDDD/market/domain/graph_test.go \
	internalDDD/market/domain/orderbook.go \
	internalDDD/market/domain/route.go \
	internalDDD/market/domain/route_index.go \
	internalDDD/market/domain/route_index_test.go \
	internalDDD/market/domain/route_test.go \
	internalDDD/market/domain/symbol.go 
do
  echo "===== $f ====="
  cat "$f"
done

for f in \
	internalDDD/market/infrastructure/binance/constants.go \
	internalDDD/market/infrastructure/binance/exchange.go \
	internalDDD/market/infrastructure/binance/exchange_mapper.go \
	internalDDD/market/infrastructure/binance/exchange_provider.go \
	internalDDD/market/infrastructure/binance/exchange_provider_test.go \
	internalDDD/market/infrastructure/binance/orderbook_dto.go 
do
  echo "===== $f ====="
  cat "$f"
done

for f in \
	internalDDD/market/infrastructure/binance/orderbook_mapper.go \
	internalDDD/market/infrastructure/binance/orderbook_mapper_test.go \
	internalDDD/market/infrastructure/binance/orderbook_provider.go \
	internalDDD/market/infrastructure/binance/orderbook_stream.go \
	internalDDD/market/infrastructure/binance/orderbook_stream_test.go 
do
  echo "===== $f ====="
  cat "$f"
done

for f in \
	internalDDD/market/infrastructure/binance/symbol_provider.go \
	internalDDD/market/infrastructure/orderbook_store.go \
	internalDDD/market/infrastructure/orderbook_store_test.go \
	internalDDD/routing/domain/edge.go \
	internalDDD/routing/domain/graph.go \
	internalDDD/routing/domain/graph_builder.go \
	internalDDD/routing/domain/route.go \
	internalDDD/routing/domain/route_finder.go 
do
  echo "===== $f ====="
  cat "$f"
done



