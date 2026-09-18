/**
 * @typedef {Object} ClientDetail
 * @property {string} id
 * @property {string} name
 * @property {string} slug
 * @property {string} billing_plan
 * @property {number} plus_licenses
 * @property {string} stripe_customer_id
 * @property {string} created_at
 * @property {string} owner
 * @property {string} email
 * @property {number} gateways_limit
 * @property {number} esp32_iot_limit
 * @property {number} waiter_app_limit
 * @property {string[]} active_modules
 */

/**
 * @typedef {Object} ClientListResponse
 * @property {ClientDetail[]} clients
 * @property {number} total
 * @property {number} page
 * @property {number} page_size
 */

export const organizations = [
    {
        id: "3b2eb519-724e-4f32-bb91-4cf15d2925b6",
        name: "La Pizza Nostra",
        slug: "pizza-nostra",
        billing_plan: "pro",
        plus_licenses: 3,
        stripe_customer_id: "cus_Q8k2m91a0b",
        created_at: "2026-02-14T10:00:00Z",
        owner: "Luciano Rossi",
        email: "luciano@pizza.it",
        gateways_limit: 10,
        esp32_iot_limit: 100,
        waiter_app_limit: 400,
        active_modules: ["Notificaciones Cloud", "Integración POS Toast", "App Meseros Cloud"]
    },
    {
        id: "e10a20cb-c31b-4b14-8f0a-ef31b818a4a5",
        name: "Tasty Burgers",
        slug: "tasty-burgers",
        billing_plan: "starter",
        plus_licenses: 0,
        stripe_customer_id: "cus_Q8k5m72x8a",
        created_at: "2026-03-22T14:30:00Z",
        owner: "John Doe",
        email: "john@tastyburgers.com",
        gateways_limit: 3,
        esp32_iot_limit: 15,
        waiter_app_limit: 25,
        active_modules: ["Notificaciones Cloud"]
    },
    {
        id: "98a12bc2-10f8-4e12-8812-7bb8c1a1fa99",
        name: "Sushi Palace",
        slug: "sushi-palace",
        billing_plan: "pro",
        plus_licenses: 1,
        stripe_customer_id: "cus_Q9a1l38z5c",
        created_at: "2026-05-01T09:15:00Z",
        owner: "Yuki Tanaka",
        email: "yuki@sushipalace.co.jp",
        gateways_limit: 10,
        esp32_iot_limit: 100,
        waiter_app_limit: 400,
        active_modules: ["Notificaciones Cloud", "Integración POS TastyIgniter"]
    },
    {
        id: "4f738a19-b00a-4712-9c12-32a76fa8bde2",
        name: "Café París",
        slug: "cafe-paris",
        billing_plan: "free",
        plus_licenses: 0,
        stripe_customer_id: "",
        created_at: "2026-06-18T16:45:00Z",
        owner: "Marie Dupont",
        email: "marie@cafeparis.fr",
        gateways_limit: 1,
        esp32_iot_limit: 5,
        waiter_app_limit: 5,
        active_modules: []
    }
];

export const branches = [
    { id: "b1", org_id: "3b2eb519-724e-4f32-bb91-4cf15d2925b6", name: "Sucursal Centro", address: "Av. Principal 123" },
    { id: "b2", org_id: "3b2eb519-724e-4f32-bb91-4cf15d2925b6", name: "Sucursal Norte", address: "Calle Bosque 456" },
    { id: "b3", org_id: "e10a20cb-c31b-4b14-8f0a-ef31b818a4a5", name: "Downtown Mall", address: "Av. de los Malls 900" },
    { id: "b4", org_id: "98a12bc2-10f8-4e12-8812-7bb8c1a1fa99", name: "Ginza Main Store", address: "4-chōme Ginza" }
];

export const hubs = [
    { id: "84d5df68-96bb-49e0-8208-f40445d4ea81", organization_id: "3b2eb519-724e-4f32-bb91-4cf15d2925b6", branch_id: "b1", type: "android_screen_gateway", settings: { pos_provider: "TastyIgniter" }, esp32_iot_count: 8, cloud_waiters: 25 },
    { id: "a52ff61d-3b7c-482a-9e6e-c802871f3a21", organization_id: "3b2eb519-724e-4f32-bb91-4cf15d2925b6", branch_id: "b2", type: "physical_hub", settings: { pos_provider: "Toast" }, esp32_iot_count: 4, cloud_waiters: 20 },
    { id: "287cf8bc-a10c-4b92-ba2e-8c381f72c448", organization_id: "e10a20cb-c31b-4b14-8f0a-ef31b818a4a5", branch_id: "b3", type: "android_screen_gateway", settings: { pos_provider: "TastyIgniter" }, esp32_iot_count: 5, cloud_waiters: 8 },
    { id: "9efc11a0-d128-4fb5-9c88-e219ba381c81", organization_id: "98a12bc2-10f8-4e12-8812-7bb8c1a1fa99", branch_id: "b4", type: "physical_hub", settings: { pos_provider: "CustomAPI" }, esp32_iot_count: 2, cloud_waiters: 15 }
];
