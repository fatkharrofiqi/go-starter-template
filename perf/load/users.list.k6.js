import http from "k6/http";
import { check } from "k6";

// Configuration test scenarios
export const options = {
	stages: [
		{ duration: "10s", target: 50 }, // Ramp up to 50 VUs in 10s
		{ duration: "50s", target: 100 }, // Ramp up to 100 VUs in 50s
		{ duration: "10s", target: 0 }, // Ramp down to 0 VUs in 10s
	],
	thresholds: {
		http_req_duration: ["p(99)<200"], // 99% of requests must be below 200ms
	},
};

// Base URL
const BASE_URL = "http://localhost:3000";

// Test credentials
const TEST_CREDENTIALS = {
	email: "test@test.com",
	password: "password",
};

// Setup function - Login once and return tokens
export function setup() {
	console.log("Starting K6 performance test...");
	console.log("Performing one-time login...");

	// Step 1: Login to get access token
	const loginResponse = http.post(
		`${BASE_URL}/api/auth/login`,
		JSON.stringify(TEST_CREDENTIALS),
		{
			headers: {
				"Content-Type": "application/json",
			},
		},
	);

	// Check login response
	const loginSuccess = check(loginResponse, {
		"setup: login status is 200": (r) => r.status === 200,
		"setup: login response has access_token": (r) => {
			try {
				const body = JSON.parse(r.body);
				// biome-ignore lint/complexity/useOptionalChain: <explanation>
				return body.data && body.data.access_token;
			} catch (e) {
				return false;
			}
		},
	});

	if (!loginSuccess || loginResponse.status !== 200) {
		console.error("Login failed in setup phase");
		return null;
	}

	// Extract access token
	let accessToken = "";
	try {
		const loginData = JSON.parse(loginResponse.body);
		accessToken = loginData.data.access_token;
	} catch (e) {
		console.error("Failed to parse login response:", e);
		return null;
	}

	console.log("Login successful! Access token obtained.");
	console.log(
		"Test scenario: 10s ramp-up to 50 VUs, 50s at 100 VUs, 10s ramp-down",
	);
	console.log("Threshold: 99th percentile response time <= 200ms");

	return {
		accessToken: accessToken,
	};
}

// Main test function - uses tokens from setup
export default function (data) {
	// Check if setup data is available
	if (!data || !data.accessToken) {
		console.error("No access token available from setup");
		return;
	}

	const { accessToken } = data;	
	// Step 1: Get user profile
	const userResponse = http.get(`${BASE_URL}/api/users/`, {
		headers: {
			Authorization: `Bearer ${accessToken}`,
		},
	});

	// Check user profile response
	const passed = check(userResponse, {
		"user profile status is 200": (r) => r.status === 200,
		"user profile response time < 200ms": (r) => r.timings.duration < 200,
		"user profile has valid response": (r) => {
			try {
				const body = JSON.parse(r.body);				
				return body.data !== undefined;
			} catch (e) {
				return false;
			}
		},
	});

	// Print details if check failed
	// if (!passed) {
	// 	console.error("Check failed for user profile");
	// 	console.log("Status:", userResponse.status);
	// 	console.log("Duration:", userResponse.timings.duration);
	// }
}

// Teardown function (runs once at the end)
export function teardown() {
	console.log("K6 performance test completed!");
}
