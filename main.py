import os
import pickle
import time

from dotenv import load_dotenv
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait

load_dotenv()

EMAIL = os.getenv("EMAIL")
PASSWORD = os.getenv("PASSWORD")
COOKIE_FILE = "cookies.pkl"

def check_if_logged_in():
    driver.get("https://typst.app/home")
    WebDriverWait(driver, 10).until(EC.presence_of_element_located((By.ID, "header-btn")))
    header_btn = driver.find_element(By.ID, "header-btn")
    return header_btn.get_attribute("href") == "https://typst.app/app/"


def setup_driver():
    options = webdriver.ChromeOptions()
    options.add_argument("--headless")  # Run in headless mode
    options.add_argument("--disable-gpu")  # Disable GPU acceleration (sometimes necessary in headless mode)
    preferences = {
                "profile.default_content_settings.popups": 0,
                "download.default_directory": os.getcwd() + os.path.sep,
                "directory_upgrade": True
            }
    options.add_experimental_option('prefs', preferences)
    driver = webdriver.Chrome(options=options)
    return driver

def load_cookies(driver):
    if os.path.exists(COOKIE_FILE):
        driver.get("https://typst.app/")
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
    load_cookies(driver)
    if check_if_logged_in():
        print("Already logged in.")
        return
    
    # Go to the signin page
    driver.get("https://typst.app/signin/")
    WebDriverWait(driver, 10).until(EC.presence_of_element_located((By.ID, "email")))

    # Fill out the email and password fields
    email_input = driver.find_element(By.ID, "email")
    email_input.send_keys(EMAIL)
    
    password_input = driver.find_element(By.ID, "password")
    password_input.send_keys(PASSWORD)
    
    # Submit the form
    password_input.send_keys(Keys.RETURN)
    WebDriverWait(driver, 10).until(EC.presence_of_element_located((By.XPATH, "//main")))
    
    # Save cookies after login
    save_cookies(driver)

def automate_task(driver):
    # 1. Go to the specified page
    driver.get("https://typst.app/team/aKj7S1kHEc96JAgoh1C5Ri")

    # Wait for the links to be present
    WebDriverWait(driver, 10).until(EC.presence_of_all_elements_located((By.CSS_SELECTOR, "main > :nth-child(2) a")))

    # 2. Get all children of <main> (specifically from children[1])
    links = driver.find_elements(By.CSS_SELECTOR, "main > :nth-child(2) a")
    hrefs = [link.get_attribute("href") for link in links]

    for href in hrefs:
        try:
            driver.get(href)

            # Wait for the 'File' button to be visible before interacting
            WebDriverWait(driver, 10).until(EC.presence_of_element_located((By.XPATH, "//button[text()='File']")))
            
            # # Switch to the newly opened tab
            driver.switch_to.window(driver.window_handles[-1])

            # Perform task on this new tab
            file_button = driver.find_element(By.XPATH, "//button[text() = 'File']")
            file_button.click()

            # Wait for the "Backup project" button to be clickable
            WebDriverWait(driver, 10).until(EC.element_to_be_clickable((By.XPATH, "//span[text()='Backup project']")))
            # Find the <span> with text "Backup project" and click its parent
            backup_span = driver.find_element(By.XPATH, "//span[text() = 'Backup project']")
            backup_span_parent = backup_span.find_element(By.XPATH, "parent::*")
            backup_span_parent.click()

            print(f"Downloaded {driver.title}")
            time.sleep(1)

        except Exception as e:
            print(f"Error encountered on link {href}: {e}")
    
    print("Task completed.")
    driver.quit()

if __name__ == "__main__":
    if not os.path.exists("work"):
        print("Please create a 'work' directory in cwd")
        exit(1)

    os.chdir("work")

    if not os.path.exists("repo"):
        os.system("git clone git@github.com:sermuns/pum2-typst-backup.git repo")
        print("Cloned repo.")

    driver = setup_driver()
    login(driver)  # Log in first (either from cookies or by logging in)
    automate_task(driver)  # After login, automate the task

    # unzip all zips into directories matching the zip names
    for file in os.listdir(os.getcwd()):
        if not file.endswith(".zip"):
            continue
        dir_name = os.path.splitext(file)[0]
        os.system(f'unzip "{file}" -d "repo/{dir_name}"')
        print(f'Unzipped {file} into repo/{dir_name}')
        os.remove(file)

    print("All zips unzipped.")
    os.chdir("repo")
    
    # Try to git add . then commit. If no changes, no problem
    os.system("git add .")
    os.system("git commit -m 'Automated commit'")
    os.system("git push")
    print("Pushed to remote.")
