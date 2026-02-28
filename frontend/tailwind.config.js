/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#132238",
        mist: "#4f6478",
        sand: "#f5efe6",
        accent: "#007f73",
        danger: "#be3a34",
      },
    },
  },
  plugins: [],
}
