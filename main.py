import os
import pickle
import time

from dotenv import load_dotenv
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.firefox.options import Options
from selenium.webdriver.common.action_chains import ActionChains
from selenium.webdriver.firefox.service import Service
from webdriver_manager.firefox import GeckoDriverManager


if not os.path.exists(".env"):
    print("No .env file found. Exiting.")
    exit(1)

load_dotenv()

EMAIL = os.getenv("EMAIL")
PASSWORD = os.getenv("PASSWORD")
COOKIE_FILE = "cookies.pkl"


WAIT_SECONDS = 20

def setup_driver():
    options = Options()
    options.add_argument("--headless")  # Run in headless mode
    options.set_preference("browser.download.folderList", 2)
    options.set_preference("browser.download.dir", os.getcwd())
    options.set_preference("browser.download.useDownloadDir", True)
    options.set_preference(
        "browser.helperApps.neverAsk.saveToDisk", "application/zip")

    return webdriver.Firefox(options=options, service=Service(GeckoDriverManager().install()))


def check_if_logged_in():
    driver.get("https://typst.app/home")
    load_cookies(driver)
    WebDriverWait(driver, WAIT_SECONDS).until(
        EC.presence_of_element_located((By.ID, "header-btn"))
    )
    header_btn = driver.find_element(By.ID, "header-btn")
    return header_btn.get_attribute("href") == "https://typst.app/app/"


def load_cookies(driver):
    if not os.path.exists(COOKIE_FILE):
        print("No cookies saved")
        return

    with open(COOKIE_FILE, "rb") as cookie_file:
        cookies = pickle.load(cookie_file)
        for cookie in cookies:
            driver.add_cookie(cookie)
    print("Cookies loaded.")
    driver.refresh()


def save_cookies(driver):
    with open(COOKIE_FILE, "wb") as cookie_file:
        pickle.dump(driver.get_cookies(), cookie_file)
    print("Cookies saved.")


def login(driver):
    if check_if_logged_in():
        print("Already logged in.")
        return

    driver.get("https://typst.app/signin/")

    WebDriverWait(driver, WAIT_SECONDS).until(
        EC.presence_of_element_located((By.ID, "email")))

    # Fill out the email and password fields
    email_input = driver.find_element(By.ID, "email")
    email_input.send_keys(EMAIL)

    password_input = driver.find_element(By.ID, "password")
    password_input.send_keys(PASSWORD)

    # Submit the form
    password_input.send_keys(Keys.RETURN)
    WebDriverWait(driver, WAIT_SECONDS).until(
        EC.presence_of_element_located((By.XPATH, "//main"))
    )

    # Save cookies after login
    save_cookies(driver)


def backup_typst(driver):
    # 1. Go to the specified page
    driver.get("https://typst.app/team/aKj7S1kHEc96JAgoh1C5Ri")

    # Wait for the links to be present
    WebDriverWait(driver, WAIT_SECONDS).until(
        EC.presence_of_all_elements_located(
            (By.CSS_SELECTOR, "main > :nth-child(2) a"))
    )

    # 2. Get all children of <main> (specifically from children[1])
    links = driver.find_elements(By.CSS_SELECTOR, "main > :nth-child(2) a")
    hrefs = [link.get_attribute("href") for link in links]

    for href in hrefs:
        try:
            driver.get(href)

            # Wait for the 'File' button to be visible before interacting
            time.sleep(5)
            WebDriverWait(driver, WAIT_SECONDS).until(
                EC.presence_of_element_located(
                    (By.XPATH, "//button[text()='File']"))
            )

            # # Switch to the newly opened tab
            driver.switch_to.window(driver.window_handles[-1])

            # Perform task on this new tab
            file_button = driver.find_element(
                By.XPATH, "//button[text() = 'File']")
            file_button.click()

            # Wait for the "Backup project" button to be clickable
            time.sleep(5)
            WebDriverWait(driver, WAIT_SECONDS).until(
                EC.element_to_be_clickable(
                    (By.XPATH, "//span[text()='Backup project']")
                )
            )

            backup_span = driver.find_element(
                By.XPATH, "//span[text() = 'Backup project']"
            )
            ActionChains(driver).move_to_element(backup_span).click().perform()

            print(f"Downloaded {driver.title}")
            time.sleep(5)

        except Exception as e:
            print(f"Error encountered on link {href}: {e}")

    print("Task completed.")
    driver.close()
    driver.quit()


if __name__ == "__main__":
    os.system("rm -rf /tmp/rust_mozprofile*")

    os.makedirs("work", exist_ok=True)
    os.chdir("work")

    # remove stray *.zip
    os.system("rm *.zip")

    if not os.path.exists("repo"):
        os.system("git clone git@github.com:sermuns/pum2-typst-backup.git repo")
        print("Cloned repo.")

    driver = setup_driver()
    login(driver)
    backup_typst(driver)

    # unzip all zips into directories matching the zip names
    for file in os.listdir(os.getcwd()):
        if not file.endswith(".zip"):
            continue
        if file.endswith("(1).zip"):
            continue
        dir_name = os.path.splitext(file)[0]
        os.system(f'unzip -o "{file}" -d "repo/{dir_name}"')
        print(f"Unzipped {file} into repo/{dir_name}")
        os.remove(file)

    print("All zips unzipped.")
    os.chdir("repo")

    os.system("git add .")
    os.system("git commit -m 'Automated commit'")
    os.system("git push -f")  # god forgive me
    print("Pushed to remote.")
