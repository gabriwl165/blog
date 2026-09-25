---
title: "Real-Time Market Data Streaming for Beginners"
date: 2026-09-24T00:00:00-03:00
draft: true
description: "Learn the market-data concepts behind a real-time system for quotes, trades, and price charts."
tags: ["distributed-systems", "market-data", "fintech"]
categories: ["engineering"]
---

Before designing a real-time market-data system, it helps to understand the information it handles. Financial markets are places where people and institutions buy and sell assets: stocks such as Apple shares, options contracts, currencies, cryptocurrencies, and more.

A market-data system does not execute those trades. Its job is to observe what is happening in the market and quickly send that information to applications, traders, charts, and algorithms.

## The small events that move a market

The basic events are quotes and trades.

A **quote** shows the current prices at which people are willing to buy or sell an asset:

- **Bid:** the highest price someone offers to pay.
- **Ask:** the lowest price someone accepts to sell for.

For example, an Apple quote might show a bid of $200.10 and an ask of $200.12. The difference between them is called the **spread**.

A **trade** is an actual completed purchase or sale. For example, 100 Apple shares may trade at $200.11.

A **tick** is a general name for one small market update, usually a quote update or a trade. During busy periods, an actively traded asset can produce many ticks every second.

## Turning ticks into chart bars

Charts often do not show every trade separately. Instead, they group trades into a time window, such as one minute, five minutes, or one day. This summary is called a **bar** or **candlestick**.

A one-minute bar records:

- **Open:** the first trade price in that minute.
- **High:** the highest trade price.
- **Low:** the lowest trade price.
- **Close:** the final trade price.
- **Volume:** the number of shares, contracts, or coins traded.

These summaries make it possible to understand price movement without displaying every individual market event.

## Where the data comes from

Market data comes from exchanges, the organizations where assets are traded. In US equities, data can be consolidated through a **Securities Information Processor (SIP)**. **OPRA** distributes US options data. Cryptocurrency exchanges commonly provide market data through WebSocket connections.

A real-time market-data product might power a live price page, an investing app watchlist, a trading-platform candlestick chart, a price alert, or an automated trading strategy.

The challenge is volume and speed. A system may receive more than 100,000 updates per second while thousands of people expect their screens to update almost immediately. The rest of this article will design a system that receives raw exchange data, translates it into a consistent format, builds chart bars, and sends each user the updates they requested.

## Define what the system must do

Before choosing technologies, turn the problem into concrete requirements. This separates the features users need from the quality targets the system must meet.

### Functional requirements

The system should:

- Ingest real-time quote and trade updates from stock, options, and cryptocurrency market-data feeds.
- Convert each feed's message format, symbol names, and timestamps into one consistent internal format.
- Let a connected client subscribe and unsubscribe from quotes, trades, and bars for selected instruments.
- Deliver a client only the updates for its subscriptions over WebSocket. Server-Sent Events (SSE) can be offered as a simpler, one-way alternative when required.
- Build one-minute, five-minute, and daily OHLCV bars from trades. OHLCV means open, high, low, close, and volume.
- Send updates for a bar while its time window is open, then mark it final after the allowed delay for late events has passed.
- Detect missing or duplicate feed messages where the data source provides sequence numbers.
- Let a client resynchronize from a fresh quote or bar snapshot after it misses stateful updates.
- Enforce a client's market-data entitlements before sending a feed that it is not allowed to receive.
- Store normalized events and finalized bars for replay, historical charts, and recovery.

### Non-functional requirements

The system should also meet measurable operational goals. The exact numbers depend on the product, but a first design could target:

| Area | Example target |
| --- | --- |
| Ingest rate | Sustain 100,000 normalized events per second, with room for short bursts. |
| Connected clients | Support thousands of concurrent WebSocket connections. |
| Internal latency | Process and route a normal event within one millisecond inside the same region. |
| Availability | Keep delivery running through an individual worker or feed-handler failure. |
| Isolation | A slow client must not increase latency or memory use without limit for other clients. |
| Ordering | Preserve a useful order for each instrument and source, rather than inventing one global order. |
| Data correctness | Detect gaps, deduplicate events where possible, and publish bar corrections for late trades. |
| Recovery | Restore current state and resume processing without losing the durable event history. |
| Observability | Measure feed health, processing lag, queue depth, dropped updates, and end-to-end latency. |

The one-millisecond goal applies to the service's internal work, not necessarily to a person's screen. A user on the public internet can experience a larger delay because of network distance, browser scheduling, and connection quality.
