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

Before choosing technologies, turn the problem into concrete requirements. For a 40-minute system-design interview, keep this list intentionally small: it defines the core flow without trying to design every production feature.

### Functional requirements

The system should:

- Ingest real-time quotes and trades from the supplied market-data feeds.
- Normalize each feed into one internal event format with a canonical instrument ID and event timestamp.
- Let a client subscribe to quotes, trades, and one-minute OHLCV bars for selected instruments over WebSocket. OHLCV means open, high, low, close, and volume.
- Build and publish the current one-minute bar from incoming trades.
- Detect feed disconnects or missing sequence numbers, then recover with a snapshot or resynchronization when the feed supports it.

### Non-functional requirements

The system should also meet a few measurable operational goals:

| Area | Example target |
| --- | --- |
| Throughput | Sustain 100,000 incoming updates per second, including short bursts. |
| Connections | Support thousands of concurrent WebSocket clients. |
| Internal latency | Process and route a normal event in under one millisecond within one region. |
| Slow clients | Keep client queues bounded; send the latest quote rather than every outdated quote. |
| Ordering | Preserve event order for one instrument where the feed provides it. |

The one-millisecond goal applies to the service's internal work, not necessarily to a person's screen. Network distance, browser scheduling, and connection quality can add noticeable delay for a user on the public internet.

Features such as market-data licensing, detailed historical storage, multi-region deployment, and late-bar corrections are important production concerns, but are reasonable follow-up discussion topics if time remains.
