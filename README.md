> this is a markdown version of the copy on the site landing page: https://youneedaspreadsheet.com

# You need a spreadsheet 📊
Get on top of your finances by tracking all of your bank accounts in Google Sheets. Having a high level view of your money is a budgeting superpower, you'll be able to analyse your spending & savings over the long term, and work out how to hit your money goals. But it'd be a pain to manually update a spreadsheet with your transactions & balances, why not let the robots do it for you? 🤖
## How much does it cost? 💸
£2 per month, billed via Stripe.

If you'd like to connect a business account check out the business page.

## Is my bank supported? 🏦
You can find the list of supported banks here.

##What is it? 💭
UK banks have a pretty cool, but under-adopted feature called 'open banking' which is a set of APIs provided by all banks that allow users & companies to read data, pay & accept payments via a common interface. This page uses Truelayer to connect to your bank and read the balance & transactions, then writes that data into a spreadsheet using the Google Sheets API.

I use this tool to have my current bank balance in a few different spreadsheets, i also feed it into BigQuery as an external table so i can make a Grafana dashboard for my daily spending & budgets, you can find a guide on how to do that here.

##Why should i trust you? 🕵️‍♀️
* This project is open source, you can find it on GitHub.
* Y.N.A.S never stores or logs any of your data & the only place the data lives is in your bank, and the spreadsheet you configure.
* The credentials used to access both google sheets and Truelayer are encrypted at rest via Google Cloud's KMS (Key Management Service).
*  Y.N.A.S doesn't read any of your other spreadsheets, only the one created by this app.
*  Y.N.A.S will only use your Google email to identify your account, send receipts, and notify you if there are any issues.
*  Your data is never shared with a third party without your direct written or verbal consent.

