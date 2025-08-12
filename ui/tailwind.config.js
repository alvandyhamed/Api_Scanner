/** @type {import('tailwindcss').Config} */
export default {
    content: ['./index.html', './src/**/*.{js,jsx,ts,tsx}'],
    theme: {
        extend: {
            fontFamily: {
                sans: ['system-ui','Segoe UI','Roboto','Vazirmatn','Helvetica','Arial','sans-serif'],
            },
            borderRadius: { '2xl': '1rem' },
        },
    },
    plugins: [],
}
