module.exports = {
  apps: [
    {
      name: "pixstore",
      script: "./server",
      interpreter: "none", // Important for binaries
      env: {
        NODE_ENV: "production",
        PORT: process.env.PORT || 8081,
        DATABASE_URL: process.env.DATABASE_URL,
        ADMIN_SECRET: process.env.ADMIN_SECRET,
        API_KEY: process.env.API_KEY,
        STORAGE_PATH: process.env.STORAGE_PATH,
      },
    },
  ],
};
