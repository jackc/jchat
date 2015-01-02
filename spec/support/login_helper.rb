module LoginHelper
  def login(email:, password:)
    visit '/#login'

    fill_in 'Email', with: email
    fill_in 'Password', with: password

    click_on 'Login'

    expect(page).to have_content 'Logout'
  end
end
