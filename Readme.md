# Project Name

Brief description or introduction of your project.

# Table of Contents

- [Installation](#installation)
  - [Install Go](#Install_Go)
  - [Install_GCC](#Install_GCC)
  - [Setting_CGO_ENABLED_To_1](#Setting_CGO_ENABLED_To_1)
- [Usage](#usage)
  - [Link](#link)
  - [Endpoints](#endpoints)
    - [/document](#Get_Document_By_Id)
    - [/add](#Add_a_Document)
    - [/del](#Delete_a_Document)
  - [Notes](#notes)

# Installation

To run this project locally, follow these steps:

## Install_Go

1. **Download Go:**
   - Visit the [official Go website](https://golang.org/dl/) and download the installer appropriate for your operating system.

   In this project, the developer used golang v1.21.12

2. **Install Go:**
   - Follow the installation instructions provided on the Go website for your specific operating system.

3. **Verify Installation:**
   - Open a terminal or command prompt and run:
     ```sh
     go version
     ```
   - This should display the installed Go version. If not, double-check your installation steps.

## Install_GCC
This step is needed for SQLite3 package
1. **Install GCC:**
   - **Linux:**
     - Use your package manager to install GCC. For example, on Debian-based systems:
       ```sh
       sudo apt-get update
       sudo apt-get install gcc
       ```
   - **MacOS:**
     - Install Xcode Command Line Tools, which includes GCC. You can install it from the App Store or run:
       ```sh
       xcode-select --install
       ```
   - **Windows:**
     - Install TDM-GCC, which includes GCC for Windows. Download the installer from [tdm-gcc website](https://jmeubank.github.io/tdm-gcc/articles/2021-05/10.3.0-release) and follow the installation instructions.

2. **Verify Installation:**
   - Open a terminal or command prompt and run:
     ```sh
     gcc --version
     ```
   - This should display the installed GCC version. If not, double-check your installation steps.

## Setting_CGO_ENABLED_To_1

1. **Set CGO_ENABLED=1:**
   - **Windows:**
     - Open Command Prompt and set the environment variable:
       ```cmd
       go env -w CGO_ENABLED=1
       ```
     - Alternatively, you can set it via System Properties:
       - Search for "Environment Variables" in the Start menu.
       - Click on "Edit the system environment variables".
       - Click on "Environment Variables..." button.
       - Under "System variables", click "New..." and add `CGO_ENABLED` with value `1`.

   - **Linux/MacOS:**
     - Open your terminal and set the environment variable:
       ```sh
       export CGO_ENABLED=1
       ```
     - Add this line to your shell profile (e.g., `.bashrc`, `.bash_profile`, `.zshrc`) to make it persistent across sessions.
   
2. **Verify CGO_ENABLED:**
   - Open a new terminal or command prompt and run:
     ```sh
     echo $CGO_ENABLED  # On Linux/MacOS
     ```
     ```cmd
     echo %CGO_ENABLED%  # On Windows
     ```
   - It should print `1`, confirming that `CGO_ENABLED` is set correctly.

Once you have completed these steps, you should be able to build and run the project locally on your machine.

# Usage
To interact with the API endpoints, follow the guidelines below:

## Link
Assuming the application is running locally, the base URL would typically be:
```
http://localhost:3456
```
Adjust the URL accordingly if the application is deployed elsewhere.

## Endpoints
1. ### Get_Document_By_Id
Returns a document from the database based on the provided ID.

- **URL:** `/document?id={id}`
- **Method:** `GET`
- **URL Parameters:**
  - `id`: ID of the document to fetch (required)
- **Success Response:**
  - **Code:** 200 OK
  - **Content:** JSON object representing the document
    ```json
    {
      "ID": "1",
      "Title": "Sample Document",
      "Description": "This is a sample document.",
      "Author": "John Doe",
      "CreatedAt": "2023-01-01",
      "XMLData": [
        "<title>Sample Document</title>",
        "<description>This is a sample document.</description>",
        "<author>John Doe</author>",
        "<creationDate>2023-01-01</creationDate>"
      ]
    }
    ```
- **Error Response:**
  - **Code:** 404 Not Found
  - **Content:** `{ "error": "Document with ID {id} not found" }`
  
2. ### Add_a_Document

Adds a new document to the database.

- **URL:** `/add`
- **Method:** `POST`
- **Request Body:**
  - XML data representing the document
  - Example:
    ```xml
    <title>New Document</title>
    <description>This is a new document.</description>
    <author>Jane Smith</author>
    <creationDate>2024-07-09</creationDate>
    ```
- **Success Response:**
  - **Code:** 201 Created
  - **Content:** None
- **Error Response:**
  - **Code:** 400 Bad Request
  - **Content:** `{ "error": "Failed to parse document: {error_message}" }`
  
3. ### Delete_a_Document

Deletes a document from the database based on the provided ID.

- **URL:** `/del?id={id}`
- **Method:** `DELETE`
- **URL Parameters:**
  - `id`: ID of the document to delete (required)
- **Success Response:**
  - **Code:** 200 OK
  - **Content:** None
- **Error Response:**
  - **Code:** 400 Bad Request
  - **Content:** `{ "error": "Failed to delete document with ID {id}: {error_message}" }`

## Notes

- Ensure that the XML data provided for adding a document adheres to the expected format.
- Handle errors gracefully based on the provided error messages.