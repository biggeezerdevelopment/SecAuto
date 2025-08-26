"""
Setup script for SecAuto Python SDK
"""

from setuptools import setup, find_packages
import os

# Read the README file
def read_readme():
    readme_path = os.path.join(os.path.dirname(__file__), 'secauto_sdk', 'README.md')
    if os.path.exists(readme_path):
        with open(readme_path, 'r', encoding='utf-8') as f:
            return f.read()
    return "SecAuto Python SDK - A comprehensive SDK for the SecAuto SOAR automation platform."

# Read requirements
def read_requirements():
    requirements_path = os.path.join(os.path.dirname(__file__), 'secauto_sdk', 'requirements.txt')
    if os.path.exists(requirements_path):
        with open(requirements_path, 'r', encoding='utf-8') as f:
            return [line.strip() for line in f if line.strip() and not line.startswith('#')]
    return ['restfly>=1.4.7', 'requests>=2.25.0', 'urllib3>=1.26.0']

setup(
    name='secauto-sdk',
    version='1.0.0',
    description='Python SDK for SecAuto SOAR automation platform',
    long_description=read_readme(),
    long_description_content_type='text/markdown',
    author='SecAuto Team',
    author_email='support@secauto.com',
    url='https://github.com/secauto/secauto-sdk',
    packages=find_packages(),
    include_package_data=True,
    install_requires=read_requirements(),
    python_requires='>=3.7',
    classifiers=[
        'Development Status :: 4 - Beta',
        'Intended Audience :: Developers',
        'License :: OSI Approved :: MIT License',
        'Operating System :: OS Independent',
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.7',
        'Programming Language :: Python :: 3.8',
        'Programming Language :: Python :: 3.9',
        'Programming Language :: Python :: 3.10',
        'Programming Language :: Python :: 3.11',
        'Programming Language :: Python :: 3.12',
        'Topic :: Software Development :: Libraries :: Python Modules',
        'Topic :: Security',
        'Topic :: System :: Systems Administration',
    ],
    keywords='secauto soar automation security orchestration api sdk',
    project_urls={
        'Bug Reports': 'https://github.com/secauto/secauto-sdk/issues',
        'Source': 'https://github.com/secauto/secauto-sdk',
        'Documentation': 'https://secauto.readthedocs.io/',
    },
    entry_points={
        'console_scripts': [
            'secauto=secauto_sdk.cli:main',
        ],
    },
)
