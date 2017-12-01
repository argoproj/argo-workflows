module.exports = {
  '/api': {
      'target': process.env.ARGO_API_URL || 'http://localhost:8001',
      'secure': false,
  }
};
