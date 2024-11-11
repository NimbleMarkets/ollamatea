# ot-timechart

`ot-timechart` plots a CSV file using [our `ntcharts` Terminal Charting library](https://github.com/NimbleMarkets/ntcharts) and performs an inference of it.

We test this quick by piping an `echo` of the CSV to `ot-timechart` via `stidin`, and using `--in` with the special filename `-`:

```
$ echo "date,num\n2024-01-01,1\n2024-02-01,2\n2024-03-01,4\n2024-04-01,8\n2024-05-01,16\n" | ot-timechart --in -
```

The CSV file should have a header row with the first column being the time.  The repo includes a sample CSV file `tests/SPY.dbeq_basic.20241109.csv.zstd`.  That is a zstd-compressed CSV file of daily SPY stock data for the past year, with data sourced from Databento Basic via this `dbn-hist-go` command.  You can learn more about DataBento and our [`dbn-go` project here](https://github.com/NimbleMarkets/dbn-go).

```
dbn-go-hist get-range  -d DBEQ.BASIC -s ohlcv-1d -t 20231109 -e 20241109 --encoding csv --sout id  -o SPY.dbeq_basic.20241109.csv SPY
```

Using that command incurs some minor cost (pennies); one can modify the query and generate larger costs, so caveat emptor.  This project and `dbn-go` are not affiliated with DataBento.

We can extract that and pass it to `ot-timechart`:

```
$ zstdcat tests/SPY.dbeq_basic.20241109.csv.zstd | gawk -F , '{$1; $8=$8/1000000000} {print $1","$8 }' > tests/SPY.2024.1109.cut.csv.zstd

$ ot-timechart --in  --in tests/SPY.2024.1109.cut.csv.zstd --title "SPY from 2023-11-09 to 2024-11-09" --braille
```
