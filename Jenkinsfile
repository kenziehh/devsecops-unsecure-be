pipeline {
    agent any

    environment {
        IMAGE_NAME = 'unsecure-cashflow-be-app'

        // Staging Ports
        STAGING_APP_PORT = '8091'
        STAGING_DB_PORT = '5443'
        STAGING_REDIS_PORT = '6390'

        // Production Ports
        PROD_APP_PORT = '8090'
        PROD_DB_PORT = '5442'
        PROD_REDIS_PORT = '6389'

        // Internal Container Ports
        CONTAINER_APP_PORT = '8080'
        CONTAINER_DB_PORT = '5432'
        CONTAINER_REDIS_PORT = '6379'

        SONAR_PROJECT_KEY = 'unsecure-cashflow-be'
    }

    options {
        buildDiscarder(logRotator(numToKeepStr: '10', daysToKeepStr: '30'))
        timestamps()
        timeout(time: 20, unit: 'MINUTES')
    }

    stages {
        stage('Checkout') {
            steps {
                git url: 'https://github.com/kenziehh/devsecops-unsecure-be.git', branch: 'main'
            }
        }

        stage('Build') {
            steps {
                script {
                    echo "🐳 Building Docker image..."
                    sh """
                        docker build \
                            --build-arg VERSION=${env.BUILD_NUMBER} \
                            -t ${IMAGE_NAME}:${env.BUILD_NUMBER} \
                            -t ${IMAGE_NAME}:latest \
                            -f Dockerfile.prod .
                        echo "✅ Image built: ${IMAGE_NAME}:${env.BUILD_NUMBER}"
                    """
                }
            }
        }

        stage('SAST - SonarQube Scan') {
            steps {
                script {
                    echo "🔍 Running SonarQube analysis..."
                    withSonarQubeEnv('SonarQube') {
                        sh """
                            sonar-scanner \
                                -Dsonar.projectKey=${SONAR_PROJECT_KEY} \
                                -Dsonar.sources=. \
                                -Dsonar.exclusions=**/vendor/**,**/*_test.go,**/testdata/**,**/mocks/** \
                                -Dsonar.go.coverage.reportPaths=coverage.out \
                                -Dsonar.tests=. \
                                -Dsonar.test.inclusions=**/*_test.go
                        """
                    }
                    echo "⏳ Waiting for SonarQube Quality Gate result..."
                    timeout(time: 5, unit: 'MINUTES') {
                        def qualityGate = waitForQualityGate()
                        if (qualityGate.status != 'OK') {
                            error "Quality gate failed: ${qualityGate.status}"
                        }
                    }
                }
            }
        }

        stage('Deploy Staging') {
            steps {
                script {
                    echo "🚀 Deploying to staging..."
                    sh """
                        # Staging environment variables
                        cat > .env.staging <<EOF
APP_PORT=${STAGING_APP_PORT}
DB_HOST=postgres
DB_PORT=${CONTAINER_DB_PORT}
DB_USER=postgres
DB_PASSWORD=staging_password
DB_NAME=cashflow_staging
REDIS_HOST=redis
REDIS_PORT=${CONTAINER_REDIS_PORT}
JWT_SECRET=staging_jwt_secret
CORS_ALLOWED_ORIGINS=http://localhost:3011,https://cashflow.nflrmvs.cloud
ENVIRONMENT=staging
EOF

                        # Staging port overrides
                        cat > docker-compose.staging.yml <<EOF
services:
  app:
    image: ${IMAGE_NAME}:latest
    container_name: unsecure-cashflow-be-staging
  postgres:
    container_name: unsecure-cashflow-db-staging
    ports: ["${STAGING_DB_PORT}:${CONTAINER_DB_PORT}"]
  redis:
    container_name: unsecure-redis-cashflow-staging
EOF

                        # Stop existing staging and remove volumes (to reset DB password)
                        docker compose -p staging -f docker-compose.prod.yml -f docker-compose.staging.yml --env-file .env.staging down -v 2>/dev/null || true

                        # Deploy staging
                        docker compose -p staging -f docker-compose.prod.yml -f docker-compose.staging.yml --env-file .env.staging up -d

                        # Wait for services
                        echo "⏳ Waiting for services..."
                        sleep 15

                        # Verify deployment
                        docker compose -p staging -f docker-compose.prod.yml -f docker-compose.staging.yml ps

                        if docker compose -p staging -f docker-compose.prod.yml -f docker-compose.staging.yml ps | grep -q "Up"; then
                            echo "✅ Staging deployed: http://localhost:${STAGING_APP_PORT}"
                        else
                            echo "⚠️ App may be restarting, checking logs..."
                            docker compose -p staging -f docker-compose.prod.yml -f docker-compose.staging.yml logs --tail=50 app
                        fi
                    """
                }
            }
        }

        stage('Image Scan Trivy') {
            steps {
                script {
                    echo "🔒 Running Trivy image scan..."
                    sh """
                        trivy image --severity HIGH,CRITICAL  \
                        --skip-dirs /usr/local/lib \
                        --exit-code 1 ${IMAGE_NAME}:latest || {
                            echo "❌ Vulnerabilities found in the image image!"
                            exit 1
                        }
                        echo "✅ No HIGH or CRITICAL vulnerabilities found."
                    """
                }
            }
        }

        stage('DAST - OWASP ZAP Scan') {
            steps {
                script {
                    echo "🛡️ Running OWASP ZAP DAST scan..."
                    sh """
                        # Run ZAP API scan
                        docker run --rm -u 0 \\
                          --network staging_unsecure_app_network \\
                          -v \$(pwd):/zap/wrk \\
                          -t zaproxy/zap-stable zap-api-scan.py \\
                          -t http://cashflow-be-staging:${STAGING_APP_PORT} \\
                          -f openapi \\
                          -r zap-report.html \\
                          -w zap-report.md \\
                          -J zap-report.json \\
                          -I
                    """
                }
            }
        }

        stage('Deploy Production') {
            steps {
                script {
                    input message: '🚀 Deploy to Production?', ok: 'Deploy', submitter: 'admin,devops'

                    echo "🚀 Deploying to production..."
                    sh """
                        # delete env file if exists
                        rm -f .env

                        # Create production .env if not exists
                        if [ ! -f .env ]; then
                            cat > .env <<EOF
APP_PORT=${PROD_APP_PORT}
DB_HOST=postgres
DB_PORT=${CONTAINER_DB_PORT}
DB_USER=postgres
DB_PASSWORD=production_password
DB_NAME=cashflow_prod
REDIS_HOST=redis
REDIS_PORT=${CONTAINER_REDIS_PORT}
JWT_SECRET=production_jwt_secret
CORS_ALLOWED_ORIGINS=http://localhost:3010,https://cashflow.nflrmvs.cloud
ENVIRONMENT=production
EOF
                        fi

                        # Stop existing production
                        docker compose -f docker-compose.prod.yml down 2>/dev/null || true

                        # Deploy production
                        docker compose -f docker-compose.prod.yml --env-file .env up -d

                        # Wait for services
                        echo "⏳ Waiting for services..."
                        sleep 15

                        # Verify deployment
                        if docker compose -f docker-compose.prod.yml ps | grep -q "Up"; then
                            echo "✅ Production deployed: http://localhost:${PROD_APP_PORT}"
                            docker compose -f docker-compose.prod.yml ps
                        else
                            echo "⚠️ Deployment issue, checking status..."
                            docker compose -f docker-compose.prod.yml ps
                            docker compose -f docker-compose.prod.yml logs --tail=30
                        fi
                    """
                }
            }
        }
    }

    post {
        always {
            sh """
                # Clean old images (keep last 5)
                docker images ${IMAGE_NAME} --format "{{.Tag}}" | \
                    grep -E '^[0-9]+\$' | sort -rn | tail -n +6 | \
                    xargs -I {} docker rmi ${IMAGE_NAME}:{} 2>/dev/null || true
                docker image prune -f
            """
        }
        success { echo "✅ Deployment completed successfully" }
        failure { echo "❌ Deployment failed" }
    }
}
