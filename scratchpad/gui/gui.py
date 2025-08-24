import sys
from PyQt6.QtWidgets import QApplication, QWidget, QLabel
from PyQt6.QtCore import Qt

if __name__ == '__main__':
    # Create the application instance
    app = QApplication(sys.argv)

    # Create a basic QWidget to serve as the main window
    window = QWidget()
    window.setWindowTitle('PyQt Hello World')
    window.setGeometry(100, 100, 300, 200) # x, y, width, height

    # Create a QLabel and set its text
    label = QLabel('Hello, PyQt!', window)
    label.setAlignment(Qt.AlignmentFlag.AlignCenter) # Center the text within the label
    label.setGeometry(0, 0, 300, 200) # Make label fill the window

    # Show the window
    window.show()

    # Start the application's event loop
    sys.exit(app.exec())
