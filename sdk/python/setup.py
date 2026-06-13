from setuptools import setup

setup(
    name="rbac-client",
    version="1.0.0",
    description="RBAC 权限管理系统 Python SDK",
    author="RBAC Team",
    py_modules=["rbac_client"],
    python_requires=">=3.8",
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
)
