|                        Metric |                Task |   Baseline |   Contender |        Diff |   Unit |    Diff % |
|------------------------------:|--------------------:|-----------:|------------:|------------:|-------:|----------:|
|       Total Young Gen GC time |                     |    0.252   |    0        |    -0.252   |      s |  -100.00% |
|      Total Young Gen GC count |                     |    3       |    0        |    -3       |        |  -100.00% |
|         Total Old Gen GC time |                     |    0       |    0        |     0       |      s |     0.00% |
|        Total Old Gen GC count |                     |    0       |    0        |     0       |        |     0.00% |
|   Total Ingest Pipeline count |                     |    0       |    0        |     0       |        |     0.00% |
|    Total Ingest Pipeline time |                     |    0       |    0        |     0       |     ms |     0.00% |
|  Total Ingest Pipeline failed |                     |    0       |    0        |     0       |        |     0.00% |
|                Min Throughput |          bulk-index | 7972.84    | 1992.87     | -5979.96    | docs/s |   -75.00% |
|               Mean Throughput |          bulk-index | 7972.84    | 2006.89     | -5965.95    | docs/s |   -74.83% |
|             Median Throughput |          bulk-index | 7972.84    | 2006.89     | -5965.95    | docs/s |   -74.83% |
|                Max Throughput |          bulk-index | 7972.84    | 2020.9      | -5951.93    | docs/s |   -74.65% |
|       50th percentile latency |          bulk-index |   14.7321  |  119.332    |   104.6     |     ms |  +710.01% |
|       90th percentile latency |          bulk-index |   29.7237  |  136.788    |   107.064   |     ms |  +360.20% |
|      100th percentile latency |          bulk-index |  115.425   |  177.497    |    62.0717  |     ms |   +53.78% |
|  50th percentile service time |          bulk-index |   14.7321  |  119.332    |   104.6     |     ms |  +710.01% |
|  90th percentile service time |          bulk-index |   29.7237  |  136.788    |   107.064   |     ms |  +360.20% |
| 100th percentile service time |          bulk-index |  115.425   |  177.497    |    62.0717  |     ms |   +53.78% |
|                    error rate |          bulk-index |    0       |    0        |     0       |      % |     0.00% |
|                Min Throughput |     match-all-query |  722.82    |   44.6273   |  -678.192   |  ops/s |   -93.83% |
|               Mean Throughput |     match-all-query |  722.82    |   46.3065   |  -676.513   |  ops/s |   -93.59% |
|             Median Throughput |     match-all-query |  722.82    |   46.4853   |  -676.334   |  ops/s |   -93.57% |
|                Max Throughput |     match-all-query |  722.82    |   47.2508   |  -675.569   |  ops/s |   -93.46% |
|       50th percentile latency |     match-all-query |    1.56206 |   42.1959   |    40.6338  |     ms | +2601.29% |
|       90th percentile latency |     match-all-query |    2.67695 |   48.9671   |    46.2902  |     ms | +1729.21% |
|       99th percentile latency |     match-all-query |    4.39318 |   65.8024   |    61.4092  |     ms | +1397.83% |
|      100th percentile latency |     match-all-query |    6.16208 |   82.6445   |    76.4824  |     ms | +1241.18% |
|  50th percentile service time |     match-all-query |    1.56206 |   42.1959   |    40.6338  |     ms | +2601.29% |
|  90th percentile service time |     match-all-query |    2.67695 |   48.9671   |    46.2902  |     ms | +1729.21% |
|  99th percentile service time |     match-all-query |    4.39318 |   65.8024   |    61.4092  |     ms | +1397.83% |
| 100th percentile service time |     match-all-query |    6.16208 |   82.6445   |    76.4824  |     ms | +1241.18% |
|                    error rate |     match-all-query |    0       |    0        |     0       |      % |     0.00% |
|                Min Throughput |   match-title-query |  692.949   | 3219.31     |  2526.36    |  ops/s |  +364.58% |
|               Mean Throughput |   match-title-query |  692.949   | 3219.31     |  2526.36    |  ops/s |  +364.58% |
|             Median Throughput |   match-title-query |  692.949   | 3219.31     |  2526.36    |  ops/s |  +364.58% |
|                Max Throughput |   match-title-query |  692.949   | 3219.31     |  2526.36    |  ops/s |  +364.58% |
|       50th percentile latency |   match-title-query |    1.34933 |    0.383729 |    -0.9656  |     ms |   -71.56% |
|       90th percentile latency |   match-title-query |    2.0363  |    0.593358 |    -1.44295 |     ms |   -70.86% |
|       99th percentile latency |   match-title-query |    4.04403 |    2.42858  |    -1.61545 |     ms |   -39.95% |
|      100th percentile latency |   match-title-query |    9.72592 |    2.91879  |    -6.80713 |     ms |   -69.99% |
|  50th percentile service time |   match-title-query |    1.34933 |    0.383729 |    -0.9656  |     ms |   -71.56% |
|  90th percentile service time |   match-title-query |    2.0363  |    0.593358 |    -1.44295 |     ms |   -70.86% |
|  99th percentile service time |   match-title-query |    4.04403 |    2.42858  |    -1.61545 |     ms |   -39.95% |
| 100th percentile service time |   match-title-query |    9.72592 |    2.91879  |    -6.80713 |     ms |   -69.99% |
|                    error rate |   match-title-query |    0       |    0        |     0       |      % |     0.00% |
|                Min Throughput |  match-phrase-query |  489.724   |  370.684    |  -119.041   |  ops/s |   -24.31% |
|               Mean Throughput |  match-phrase-query |  489.724   |  370.684    |  -119.041   |  ops/s |   -24.31% |
|             Median Throughput |  match-phrase-query |  489.724   |  370.684    |  -119.041   |  ops/s |   -24.31% |
|                Max Throughput |  match-phrase-query |  489.724   |  370.684    |  -119.041   |  ops/s |   -24.31% |
|       50th percentile latency |  match-phrase-query |    1.42254 |    4.38252  |     2.95998 |     ms |  +208.08% |
|       90th percentile latency |  match-phrase-query |    3.97606 |    8.2713   |     4.29524 |     ms |  +108.03% |
|       99th percentile latency |  match-phrase-query |   15.3382  |   14.1873   |    -1.15094 |     ms |    -7.50% |
|      100th percentile latency |  match-phrase-query |  161.899   |   22.4612   |  -139.438   |     ms |   -86.13% |
|  50th percentile service time |  match-phrase-query |    1.42254 |    4.38252  |     2.95998 |     ms |  +208.08% |
|  90th percentile service time |  match-phrase-query |    3.97606 |    8.2713   |     4.29524 |     ms |  +108.03% |
|  99th percentile service time |  match-phrase-query |   15.3382  |   14.1873   |    -1.15094 |     ms |    -7.50% |
| 100th percentile service time |  match-phrase-query |  161.899   |   22.4612   |  -139.438   |     ms |   -86.13% |
|                    error rate |  match-phrase-query |    0       |    0        |     0       |      % |     0.00% |
|                Min Throughput | term-category-query | 1101.15    | 8667.47     |  7566.32    |  ops/s |  +687.13% |
|               Mean Throughput | term-category-query | 1101.15    | 8667.47     |  7566.32    |  ops/s |  +687.13% |
|             Median Throughput | term-category-query | 1101.15    | 8667.47     |  7566.32    |  ops/s |  +687.13% |
|                Max Throughput | term-category-query | 1101.15    | 8667.47     |  7566.32    |  ops/s |  +687.13% |
|       50th percentile latency | term-category-query |    1.08813 |    0.111354 |    -0.97677 |     ms |   -89.77% |
|       90th percentile latency | term-category-query |    1.75409 |    0.16265  |    -1.59144 |     ms |   -90.73% |
|       99th percentile latency | term-category-query |    2.79246 |    0.310574 |    -2.48188 |     ms |   -88.88% |
|      100th percentile latency | term-category-query |    3.20829 |    0.457249 |    -2.75104 |     ms |   -85.75% |
|  50th percentile service time | term-category-query |    1.08813 |    0.111354 |    -0.97677 |     ms |   -89.77% |
|  90th percentile service time | term-category-query |    1.75409 |    0.16265  |    -1.59144 |     ms |   -90.73% |
|  99th percentile service time | term-category-query |    2.79246 |    0.310574 |    -2.48188 |     ms |   -88.88% |
| 100th percentile service time | term-category-query |    3.20829 |    0.457249 |    -2.75104 |     ms |   -85.75% |
|                    error rate | term-category-query |    0       |    0        |     0       |      % |     0.00% |
|                Min Throughput | numeric-range-query |  886.621   |  103.592    |  -783.029   |  ops/s |   -88.32% |
|               Mean Throughput | numeric-range-query |  886.621   |  122.056    |  -764.565   |  ops/s |   -86.23% |
|             Median Throughput | numeric-range-query |  886.621   |  125.887    |  -760.734   |  ops/s |   -85.80% |
|                Max Throughput | numeric-range-query |  886.621   |  132.858    |  -753.763   |  ops/s |   -85.02% |
|       50th percentile latency | numeric-range-query |    1.24356 |   13.0916   |    11.8481  |     ms |  +952.75% |
|       90th percentile latency | numeric-range-query |    2.52055 |   16.0623   |    13.5417  |     ms |  +537.25% |
|       99th percentile latency | numeric-range-query |    3.81466 |   24.5283   |    20.7136  |     ms |  +543.00% |
|      100th percentile latency | numeric-range-query |    5.01833 |   37.462    |    32.4436  |     ms |  +646.50% |
|  50th percentile service time | numeric-range-query |    1.24356 |   13.0916   |    11.8481  |     ms |  +952.75% |
|  90th percentile service time | numeric-range-query |    2.52055 |   16.0623   |    13.5417  |     ms |  +537.25% |
|  99th percentile service time | numeric-range-query |    3.81466 |   24.5283   |    20.7136  |     ms |  +543.00% |
| 100th percentile service time | numeric-range-query |    5.01833 |   37.462    |    32.4436  |     ms |  +646.50% |
|                    error rate | numeric-range-query |    0       |    0        |     0       |      % |     0.00% |
|                Min Throughput | bool-compound-query |  608.059   | 7814.89     |  7206.83    |  ops/s | +1185.22% |
|               Mean Throughput | bool-compound-query |  608.059   | 7814.89     |  7206.83    |  ops/s | +1185.22% |
|             Median Throughput | bool-compound-query |  608.059   | 7814.89     |  7206.83    |  ops/s | +1185.22% |
|                Max Throughput | bool-compound-query |  608.059   | 7814.89     |  7206.83    |  ops/s | +1185.22% |
|       50th percentile latency | bool-compound-query |    1.61633 |    0.121936 |    -1.4944  |     ms |   -92.46% |
|       90th percentile latency | bool-compound-query |    3.35674 |    0.171712 |    -3.18503 |     ms |   -94.88% |
|       99th percentile latency | bool-compound-query |    8.43185 |    0.310495 |    -8.12135 |     ms |   -96.32% |
|      100th percentile latency | bool-compound-query |   40.3385  |    0.513874 |   -39.8246  |     ms |   -98.73% |
|  50th percentile service time | bool-compound-query |    1.61633 |    0.121936 |    -1.4944  |     ms |   -92.46% |
|  90th percentile service time | bool-compound-query |    3.35674 |    0.171712 |    -3.18503 |     ms |   -94.88% |
|  99th percentile service time | bool-compound-query |    8.43185 |    0.310495 |    -8.12135 |     ms |   -96.32% |
| 100th percentile service time | bool-compound-query |   40.3385  |    0.513874 |   -39.8246  |     ms |   -98.73% |
|                    error rate | bool-compound-query |    0       |    0        |     0       |      % |     0.00% |
