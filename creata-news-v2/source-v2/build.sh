set -e
cd "$(dirname "$0")"
node build-data.js
npx esbuild src/main.jsx --bundle --minify --format=iife --target=es2019 --jsx=automatic --define:process.env.NODE_ENV='"production"' --outfile=../site/assets/app.js --log-level=warning
npx tailwindcss -c tailwind.config.js -i src/styles.css -o ../site/assets/app.css --minify 2>/dev/null
