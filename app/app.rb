class Wallpapers < Padrino::Application
  register Padrino::Rendering
  register Padrino::Helpers
  register Padrino::Cache
  enable :caching
  set :logging, true            # Logging in STDOUT for development and file for production (default only for development)
end
