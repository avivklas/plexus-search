|                        Metric |                Task |    Baseline |   Contender |        Diff |   Unit |   Diff % |
|------------------------------:|--------------------:|------------:|------------:|------------:|-------:|---------:|
|       Total Young Gen GC time |                     |    0        |    0        |     0       |      s |    0.00% |
|      Total Young Gen GC count |                     |    0        |    0        |     0       |        |    0.00% |
|         Total Old Gen GC time |                     |    0        |    0        |     0       |      s |    0.00% |
|        Total Old Gen GC count |                     |    0        |    0        |     0       |        |    0.00% |
|   Total Ingest Pipeline count |                     |    0        |    0        |     0       |        |    0.00% |
|    Total Ingest Pipeline time |                     |    0        |    0        |     0       |     ms |    0.00% |
|  Total Ingest Pipeline failed |                     |    0        |    0        |     0       |        |    0.00% |
|                Min Throughput |          bulk-index | 2206.1      | 2140.36     |   -65.737   | docs/s |   -2.98% |
|               Mean Throughput |          bulk-index | 2347.46     | 2206.83     |  -140.633   | docs/s |   -5.99% |
|             Median Throughput |          bulk-index | 2347.46     | 2206.83     |  -140.633   | docs/s |   -5.99% |
|                Max Throughput |          bulk-index | 2488.83     | 2273.3      |  -215.529   | docs/s |   -8.66% |
|       50th percentile latency |          bulk-index |  118.595    |  119.943    |     1.34825 |     ms |   +1.14% |
|       90th percentile latency |          bulk-index |  135.729    |  135.135    |    -0.59363 |     ms |   -0.44% |
|      100th percentile latency |          bulk-index |  144.82     |  143.652    |    -1.16762 |     ms |   -0.81% |
|  50th percentile service time |          bulk-index |  118.595    |  119.943    |     1.34825 |     ms |   +1.14% |
|  90th percentile service time |          bulk-index |  135.729    |  135.135    |    -0.59363 |     ms |   -0.44% |
| 100th percentile service time |          bulk-index |  144.82     |  143.652    |    -1.16762 |     ms |   -0.81% |
|                    error rate |          bulk-index |    0        |    0        |     0       |      % |    0.00% |
|                Min Throughput |     match-all-query |   46.7647   |   43.9272   |    -2.83748 |  ops/s |   -6.07% |
|               Mean Throughput |     match-all-query |   47.2791   |   45.5876   |    -1.69145 |  ops/s |   -3.58% |
|             Median Throughput |     match-all-query |   47.3207   |   44.8587   |    -2.46198 |  ops/s |   -5.20% |
|                Max Throughput |     match-all-query |   48.0695   |   48.0475   |    -0.02199 |  ops/s |   -0.05% |
|       50th percentile latency |     match-all-query |   42.4919   |   43.7146   |     1.22267 |     ms |   +2.88% |
|       90th percentile latency |     match-all-query |   48.3577   |   56.0073   |     7.64954 |     ms |  +15.82% |
|       99th percentile latency |     match-all-query |   59.0105   |  102.738    |    43.7279  |     ms |  +74.10% |
|      100th percentile latency |     match-all-query |  147.689    |  136.644    |   -11.0451  |     ms |   -7.48% |
|  50th percentile service time |     match-all-query |   42.4919   |   43.7146   |     1.22267 |     ms |   +2.88% |
|  90th percentile service time |     match-all-query |   48.3577   |   56.0073   |     7.64954 |     ms |  +15.82% |
|  99th percentile service time |     match-all-query |   59.0105   |  102.738    |    43.7279  |     ms |  +74.10% |
| 100th percentile service time |     match-all-query |  147.689    |  136.644    |   -11.0451  |     ms |   -7.48% |
|                    error rate |     match-all-query |    0        |    0        |     0       |      % |    0.00% |
|                Min Throughput |   match-title-query | 2894.66     | 2684.7      |  -209.965   |  ops/s |   -7.25% |
|               Mean Throughput |   match-title-query | 2894.66     | 2684.7      |  -209.965   |  ops/s |   -7.25% |
|             Median Throughput |   match-title-query | 2894.66     | 2684.7      |  -209.965   |  ops/s |   -7.25% |
|                Max Throughput |   match-title-query | 2894.66     | 2684.7      |  -209.965   |  ops/s |   -7.25% |
|       50th percentile latency |   match-title-query |    0.401563 |    0.404104 |     0.00254 |     ms |   +0.63% |
|       90th percentile latency |   match-title-query |    0.713278 |    0.756487 |     0.04321 |     ms |   +6.06% |
|       99th percentile latency |   match-title-query |    1.67582  |    3.32224  |     1.64642 |     ms |  +98.25% |
|      100th percentile latency |   match-title-query |    2.10312  |   12.7013   |    10.5982  |     ms | +503.92% |
|  50th percentile service time |   match-title-query |    0.401563 |    0.404104 |     0.00254 |     ms |   +0.63% |
|  90th percentile service time |   match-title-query |    0.713278 |    0.756487 |     0.04321 |     ms |   +6.06% |
|  99th percentile service time |   match-title-query |    1.67582  |    3.32224  |     1.64642 |     ms |  +98.25% |
| 100th percentile service time |   match-title-query |    2.10312  |   12.7013   |    10.5982  |     ms | +503.92% |
|                    error rate |   match-title-query |    0        |    0        |     0       |      % |    0.00% |
|                Min Throughput |  match-phrase-query |  266.381    |  353.308    |    86.9274  |  ops/s |  +32.63% |
|               Mean Throughput |  match-phrase-query |  279.521    |  353.308    |    73.7874  |  ops/s |  +26.40% |
|             Median Throughput |  match-phrase-query |  279.521    |  353.308    |    73.7874  |  ops/s |  +26.40% |
|                Max Throughput |  match-phrase-query |  292.661    |  353.308    |    60.6474  |  ops/s |  +20.72% |
|       50th percentile latency |  match-phrase-query |    5.16069  |    4.58489  |    -0.57579 |     ms |  -11.16% |
|       90th percentile latency |  match-phrase-query |    9.19599  |    8.50066  |    -0.69532 |     ms |   -7.56% |
|       99th percentile latency |  match-phrase-query |   14.4726   |   14.9855   |     0.51284 |     ms |   +3.54% |
|      100th percentile latency |  match-phrase-query |   29.3007   |   25.2428   |    -4.05783 |     ms |  -13.85% |
|  50th percentile service time |  match-phrase-query |    5.16069  |    4.58489  |    -0.57579 |     ms |  -11.16% |
|  90th percentile service time |  match-phrase-query |    9.19599  |    8.50066  |    -0.69532 |     ms |   -7.56% |
|  99th percentile service time |  match-phrase-query |   14.4726   |   14.9855   |     0.51284 |     ms |   +3.54% |
| 100th percentile service time |  match-phrase-query |   29.3007   |   25.2428   |    -4.05783 |     ms |  -13.85% |
|                    error rate |  match-phrase-query |    0        |    0        |     0       |      % |    0.00% |
|                Min Throughput | term-category-query | 7891.64     | 6814.19     | -1077.46    |  ops/s |  -13.65% |
|               Mean Throughput | term-category-query | 7891.64     | 6814.19     | -1077.46    |  ops/s |  -13.65% |
|             Median Throughput | term-category-query | 7891.64     | 6814.19     | -1077.46    |  ops/s |  -13.65% |
|                Max Throughput | term-category-query | 7891.64     | 6814.19     | -1077.46    |  ops/s |  -13.65% |
|       50th percentile latency | term-category-query |    0.129959 |    0.135041 |     0.00508 |     ms |   +3.91% |
|       90th percentile latency | term-category-query |    0.164729 |    0.173457 |     0.00873 |     ms |   +5.30% |
|       99th percentile latency | term-category-query |    0.275983 |    0.288456 |     0.01247 |     ms |   +4.52% |
|      100th percentile latency | term-category-query |    0.479874 |    0.473415 |    -0.00646 |     ms |   -1.35% |
|  50th percentile service time | term-category-query |    0.129959 |    0.135041 |     0.00508 |     ms |   +3.91% |
|  90th percentile service time | term-category-query |    0.164729 |    0.173457 |     0.00873 |     ms |   +5.30% |
|  99th percentile service time | term-category-query |    0.275983 |    0.288456 |     0.01247 |     ms |   +4.52% |
| 100th percentile service time | term-category-query |    0.479874 |    0.473415 |    -0.00646 |     ms |   -1.35% |
|                    error rate | term-category-query |    0        |    0        |     0       |      % |    0.00% |
|                Min Throughput | numeric-range-query |  114.662    |  115.451    |     0.7893  |  ops/s |   +0.69% |
|               Mean Throughput | numeric-range-query |  120.27     |  119.937    |    -0.33306 |  ops/s |   -0.28% |
|             Median Throughput | numeric-range-query |  121.777    |  119.483    |    -2.29435 |  ops/s |   -1.88% |
|                Max Throughput | numeric-range-query |  124.371    |  125.331    |     0.95996 |  ops/s |   +0.77% |
|       50th percentile latency | numeric-range-query |   13.7417   |   14.5383   |     0.7966  |     ms |   +5.80% |
|       90th percentile latency | numeric-range-query |   17.4772   |   16.9216   |    -0.55566 |     ms |   -3.18% |
|       99th percentile latency | numeric-range-query |   22.6835   |   22.3246   |    -0.35892 |     ms |   -1.58% |
|      100th percentile latency | numeric-range-query |   27.5596   |   29.9475   |     2.38792 |     ms |   +8.66% |
|  50th percentile service time | numeric-range-query |   13.7417   |   14.5383   |     0.7966  |     ms |   +5.80% |
|  90th percentile service time | numeric-range-query |   17.4772   |   16.9216   |    -0.55566 |     ms |   -3.18% |
|  99th percentile service time | numeric-range-query |   22.6835   |   22.3246   |    -0.35892 |     ms |   -1.58% |
| 100th percentile service time | numeric-range-query |   27.5596   |   29.9475   |     2.38792 |     ms |   +8.66% |
|                    error rate | numeric-range-query |    0        |    0        |     0       |      % |    0.00% |
|                Min Throughput | bool-compound-query | 7093.3      | 7402.76     |   309.458   |  ops/s |   +4.36% |
|               Mean Throughput | bool-compound-query | 7093.3      | 7402.76     |   309.458   |  ops/s |   +4.36% |
|             Median Throughput | bool-compound-query | 7093.3      | 7402.76     |   309.458   |  ops/s |   +4.36% |
|                Max Throughput | bool-compound-query | 7093.3      | 7402.76     |   309.458   |  ops/s |   +4.36% |
|       50th percentile latency | bool-compound-query |    0.138124 |    0.127791 |    -0.01033 |     ms |   -7.48% |
|       90th percentile latency | bool-compound-query |    0.164746 |    0.154133 |    -0.01061 |     ms |   -6.44% |
|       99th percentile latency | bool-compound-query |    0.281491 |    0.245352 |    -0.03614 |     ms |  -12.84% |
|      100th percentile latency | bool-compound-query |    0.431832 |    0.45425  |     0.02242 |     ms |   +5.19% |
|  50th percentile service time | bool-compound-query |    0.138124 |    0.127791 |    -0.01033 |     ms |   -7.48% |
|  90th percentile service time | bool-compound-query |    0.164746 |    0.154133 |    -0.01061 |     ms |   -6.44% |
|  99th percentile service time | bool-compound-query |    0.281491 |    0.245352 |    -0.03614 |     ms |  -12.84% |
| 100th percentile service time | bool-compound-query |    0.431832 |    0.45425  |     0.02242 |     ms |   +5.19% |
|                    error rate | bool-compound-query |    0        |    0        |     0       |      % |    0.00% |
