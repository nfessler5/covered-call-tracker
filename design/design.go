package design

import (
	. "goa.design/goa/v3/dsl"
	. "goa.design/goa/v3/eval"
)

// API Level Metadata
var _ = API("covered_call_tracker", func() {
	Title("Covered Call Tracker API")
	Description("API for tracking underlying stock cost basis, option positions, and contract rolls.")
	Server("tracker", func() {
		Host("localhost", func() {
			URI("http://localhost:8080")
		})
	})
})

var _ = Service("scanner", func() {
	Description("In-app scanning engine service")

	Method("scan", func() {
		Payload(func() {
			Field(1, "tickers", ArrayOf(String), "List of tickers to scan")
			Field(2, "min_annualized_yield", Float64, "Minimum annual return yield %")
			Required("tickers")
		})
		Result(ArrayOf(Candidate))
		HTTP(func() {
			POST("/api/v1/scanner/run")
			Response(StatusOK)
		})
	})
})

var Candidate = Type("Candidate", func() {
	Field(1, "ticker", String)
	Field(2, "stock_price", Float64)
	Field(3, "strike_price", Float64)
	Field(4, "expiration_date", String)
	Field(5, "premium", Float64)
	Field(6, "annualized_yield", Float64)
})

// Data Models
var UnderlyingStock = Type("UnderlyingStock", func() {
	Description("A stock held in the portfolio for selling covered calls.")
	Field(1, "id", Int, "Unique ID", func() { Example(1) })
	Field(2, "ticker", String, "Stock Ticker Symbol", func() { Example("SOXL") })
	Field(3, "company_name", String, "Company Name", func() { Example("Direxion Semiconductor 3x") })
	Field(4, "shares_owned", Int, "Total shares owned", func() { Example(100) })
	Field(5, "effective_cost_basis", Float64, "Effective cost basis per share after options premiums", func() { Example(28.50) })
	Required("ticker", "company_name", "shares_owned", "effective_cost_basis")
})

var CoveredCallPosition = Type("CoveredCallPosition", func() {
	Description("An open or historical covered call options contract.")
	Field(1, "id", Int, "Unique Position ID", func() { Example(10) })
	Field(2, "underlying_id", Int, "ID of the underlying stock", func() { Example(1) })
	Field(3, "strike_price", Float64, "Option Strike Price", func() { Example(30.00) })
	Field(4, "expiration_date", String, "Expiration Date (YYYY-MM-DD)", func() { Example("2026-09-18") })
	Field(5, "premium_collected", Float64, "Total premium collected for the contracts", func() { Example(150.00) })
	Field(6, "contracts_count", Int, "Number of contracts sold", func() { Example(1) })
	Field(7, "status", String, "Status: OPEN, EXPIRED, ASSIGNED, ROLLED, CLOSED", func() { Example("OPEN") })
	Required("underlying_id", "strike_price", "expiration_date", "premium_collected", "contracts_count", "status")
})

var RollPositionPayload = Type("RollPositionPayload", func() {
	Description("Parameters required to roll an existing position into a new expiration and strike.")
	Field(1, "new_strike_price", Float64, "New strike price", func() { Example(32.00) })
	Field(2, "new_expiration_date", String, "New expiration date (YYYY-MM-DD)", func() { Example("2026-09-25") })
	Field(3, "net_credit", Float64, "Net credit collected (or debit paid) during the roll", func() { Example(45.00) })
	Required("new_strike_price", "new_expiration_date", "net_credit")
})

// Service Endpoints
var _ = Service("positions", func() {
	Description("Service for managing covered call positions.")

	Method("list", func() {
		Description("List active or historical covered call positions.")
		Payload(func() {
			Field(1, "status", String, "Filter by status (OPEN, EXPIRED, ASSIGNED, ROLLED, CLOSED)")
		})
		Result(ArrayOf(CoveredCallPosition))
		HTTP(func() {
			GET("/positions")
			Param("status")
			Response(StatusOK)
		})
	})

	Method("create", func() {
		Description("Open a new covered call position.")
		Payload(CoveredCallPosition)
		Result(CoveredCallPosition)
		HTTP(func() {
			POST("/positions")
			Response(StatusCreated)
		})
	})

	Method("roll", func() {
		Description("Close an existing position as ROLLED and create a new open position in one atomic transaction.")
		Payload(func() {
			Field(1, "id", Int, "ID of the position being rolled")
			Field(2, "roll_details", RollPositionPayload, "New contract details")
			Required("id", "roll_details")
		})
		Result(CoveredCallPosition)
		HTTP(func() {
			POST("/positions/{id}/roll")
			Response(StatusOK)
		})
	})
})
