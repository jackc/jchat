require 'spec_helper'

feature 'Chat' do
  scenario 'Posting a text message' do
    create_user name: "poster", email: "poster@example.com"
    login email: "poster@example.com", password: "password"

    reader = create_user

#     binding.pry

#     session = Capybara::Session.new(:webkit, my_rack_app)
# session.within("//form[@id='session']") do
#   session.fill_in 'Email', :with => 'user@example.com'
#   session.fill_in 'Password', :with => 'password'
# end
# session.click_button 'Sign in'
  end
end
